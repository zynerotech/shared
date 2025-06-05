package kafka

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/scram"
	"github.com/zynerotech/shared/transport"
)

type KafkaProducer struct {
	writer       *kafka.Writer
	defaultTopic string
	metrics      transport.Metrics
	mu           sync.RWMutex
	closed       bool
}

// NewProducer создает нового KafkaProducer на основе предоставленной конфигурации.
func NewProducer(cfg Config) (*KafkaProducer, error) {
	sharedTransport := &kafka.Transport{}
	if cfg.SASL.Enabled {
		mechanism, err := scram.Mechanism(scram.SHA512, cfg.SASL.Username, cfg.SASL.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to create SASL mechanism: %w", err)
		}
		sharedTransport.SASL = mechanism
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Balancer:     &kafka.Hash{},
		Transport:    sharedTransport,
		BatchSize:    cfg.Producer.BatchSize,
		BatchTimeout: cfg.Producer.BatchTimeout,
		RequiredAcks: kafka.RequiredAcks(cfg.Producer.RequiredAcks),
		Compression:  cfg.Producer.GetCompressionCodec(),
	}

	producer := &KafkaProducer{
		writer:       writer,
		defaultTopic: cfg.Producer.Topic,
		metrics:      &transport.NoOpMetrics{}, // По умолчанию no-op метрики
	}

	// Обновляем метрики активных producer
	producer.metrics.SetActiveProducers(1)

	return producer, nil
}

// SetMetrics устанавливает интерфейс метрик
func (p *KafkaProducer) SetMetrics(metrics transport.Metrics) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.metrics = metrics
}

func (p *KafkaProducer) Publish(ctx context.Context, topic, key string, value []byte) error {
	start := time.Now()

	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return fmt.Errorf("producer is closed")
	}

	t := p.defaultTopic
	if topic != "" {
		t = topic
	}

	metrics := p.metrics
	p.mu.RUnlock()

	// Измеряем время публикации
	defer func() {
		metrics.RecordPublishTime(t, time.Since(start))
	}()

	err := p.writer.WriteMessages(ctx, kafka.Message{
		Topic: t,
		Key:   []byte(key),
		Value: value,
	})

	// Записываем метрики результата
	if err != nil {
		metrics.IncMessagesSent(t, "error")
		return err
	}

	metrics.IncMessagesSent(t, "success")
	return nil
}

// Close выполняет graceful shutdown producer
func (p *KafkaProducer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	log.Info().Msg("Closing producer...")

	// Обновляем метрики перед закрытием
	p.metrics.SetActiveProducers(0)

	// Закрываем writer, это дождется отправки всех буферизованных сообщений
	if err := p.writer.Close(); err != nil {
		log.Error().Err(err).Msg("Error closing Kafka writer")
		return fmt.Errorf("failed to close writer: %w", err)
	}

	p.closed = true
	log.Info().Msg("Producer closed successfully")
	return nil
}
