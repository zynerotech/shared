package kafka

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"gitlab.com/zynero/shared/transport"

	json "github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader         *kafka.Reader
	handler        transport.Handler
	retryProcessor *RetryProcessor
	metrics        transport.Metrics
	topic          string

	// Каналы для graceful shutdown
	stopCh    chan struct{}
	doneCh    chan struct{}
	mu        sync.RWMutex
	isRunning bool
}

func NewConsumer(cfg Config, topic string, handler transport.Handler) *Consumer {
	consumer := &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        cfg.Brokers,
			Topic:          topic,
			GroupID:        cfg.Consumer.GroupID,
			MinBytes:       cfg.Consumer.MinBytes,
			MaxBytes:       cfg.Consumer.MaxBytes,
			MaxWait:        cfg.Consumer.MaxWait,
			CommitInterval: 0,
		}),
		handler: handler,
		topic:   topic,
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
		metrics: &transport.NoOpMetrics{}, // По умолчанию no-op метрики
	}

	// Создаем retry processor если настроена надежность
	if cfg.Reliability.RetryCount > 0 || cfg.Reliability.DLQEnabled {
		// Для DLQ нужен producer
		if cfg.Reliability.DLQEnabled && cfg.Reliability.DLQTopic != "" {
			dlqProducer, err := NewProducer(cfg)
			if err != nil {
				log.Error().Err(err).Msg("Failed to create DLQ producer, disabling retry")
			} else {
				consumer.retryProcessor = NewRetryProcessor(cfg.Reliability, dlqProducer)
			}
		}
	}

	return consumer
}

// SetMetrics устанавливает интерфейс метрик
func (c *Consumer) SetMetrics(metrics transport.Metrics) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics = metrics

	// Устанавливаем метрики и для retry processor
	if c.retryProcessor != nil {
		c.retryProcessor.SetMetrics(metrics)
	}
}

// Run запускает consumer и блокирует выполнение до получения сигнала остановки
func (c *Consumer) Run(ctx context.Context) error {
	c.mu.Lock()
	if c.isRunning {
		c.mu.Unlock()
		return fmt.Errorf("consumer is already running")
	}
	c.isRunning = true
	c.mu.Unlock()

	// Обновляем метрики активных consumer
	c.metrics.SetActiveConsumers(1)

	defer func() {
		c.mu.Lock()
		c.isRunning = false
		c.mu.Unlock()
		close(c.doneCh)

		// Обновляем метрики
		c.metrics.SetActiveConsumers(0)

		log.Info().Msg("Consumer stopped")
	}()

	log.Info().Msg("Starting consumer")

	// Создаем контекст с отменой для внутреннего использования
	consumerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Горутина для обработки сигнала остановки
	go func() {
		select {
		case <-c.stopCh:
			log.Info().Msg("Received stop signal")
			cancel()
		case <-ctx.Done():
			log.Info().Msg("Context cancelled")
			cancel()
		}
	}()

	return c.processMessages(consumerCtx)
}

// Stop инициирует graceful shutdown
func (c *Consumer) Stop() {
	c.mu.RLock()
	if !c.isRunning {
		c.mu.RUnlock()
		return
	}
	c.mu.RUnlock()

	log.Info().Msg("Stopping consumer...")
	close(c.stopCh)
}

// Wait ожидает завершения работы consumer с таймаутом
func (c *Consumer) Wait(timeout time.Duration) error {
	select {
	case <-c.doneCh:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("consumer shutdown timeout after %v", timeout)
	}
}

// Close освобождает ресурсы
func (c *Consumer) Close() error {
	c.Stop()

	// Ждем завершения с таймаутом
	if err := c.Wait(30 * time.Second); err != nil {
		log.Warn().Err(err).Msg("Consumer did not stop gracefully, forcing close")
	}

	if err := c.reader.Close(); err != nil {
		log.Error().Err(err).Msg("Error closing Kafka reader")
		return fmt.Errorf("failed to close reader: %w", err)
	}

	log.Info().Msg("Consumer closed successfully")
	return nil
}

// processMessages основной цикл обработки сообщений
func (c *Consumer) processMessages(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Context cancelled, stopping message processing")
			return nil
		default:
			// Устанавливаем таймаут для чтения сообщений
			readCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			msg, err := c.reader.ReadMessage(readCtx)
			cancel()

			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					// Проверяем, не отменен ли основной контекст
					select {
					case <-ctx.Done():
						return nil
					default:
						continue // Таймаут чтения, продолжаем
					}
				}
				log.Error().Err(err).Msg("Error reading message")
				continue
			}

			// Метрика получения сообщения
			c.metrics.IncMessagesReceived(c.topic, msg.Partition)

			if err := c.processMessage(ctx, msg); err != nil {
				log.Error().
					Err(err).
					Str("topic", msg.Topic).
					Int("partition", msg.Partition).
					Int64("offset", msg.Offset).
					Msg("Failed to process message")

				// Метрика ошибки обработки
				c.metrics.IncMessagesProcessed(c.topic, "error")

				// В случае ошибки всё равно коммитим, так как retry/DLQ уже обработаны
				if commitErr := c.reader.CommitMessages(ctx, msg); commitErr != nil {
					log.Error().Err(commitErr).Msg("Failed to commit message after processing error")
				}
				continue
			}

			// Метрика успешной обработки
			c.metrics.IncMessagesProcessed(c.topic, "success")

			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				log.Error().Err(err).Msg("Failed to commit message")
				continue
			}
		}
	}
}

func (c *Consumer) processMessage(ctx context.Context, msg kafka.Message) error {
	start := time.Now()
	defer func() {
		// Записываем время обработки
		c.metrics.RecordProcessingTime(c.topic, time.Since(start))
	}()

	// Если есть retry processor, используем его
	if c.retryProcessor != nil {
		return c.retryProcessor.ProcessWithRetry(ctx, msg, c.handler)
	}

	// Иначе используем простую обработку
	var envelope transport.Envelope
	if err := json.Unmarshal(msg.Value, &envelope); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	if err := c.handler.Handle(ctx, envelope); err != nil {
		return fmt.Errorf("handler failed: %w", err)
	}

	return nil
}
