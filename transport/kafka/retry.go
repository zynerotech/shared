package kafka

import (
	"context"
	"fmt"
	"strconv"
	"time"

	json "github.com/bytedance/sonic"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
	"gitlab.com/zynero/shared/transport"
)

// RetryableError represents an error that may or may not be retried.
type RetryableError struct {
	Err        error
	Retryable  bool
	RetryAfter time.Duration
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// NewRetryableError creates a new RetryableError instance.
func NewRetryableError(err error, retryable bool) *RetryableError {
	return &RetryableError{
		Err:       err,
		Retryable: retryable,
	}
}

// RetryProcessor handles retry logic for messages.
type RetryProcessor struct {
	config   ReliabilityConfig
	producer transport.Producer
	dlqTopic string
	metrics  transport.Metrics
}

// NewRetryProcessor creates a new processor for retries.
func NewRetryProcessor(config ReliabilityConfig, producer transport.Producer) *RetryProcessor {
	return &RetryProcessor{
		config:   config,
		producer: producer,
		dlqTopic: config.DLQTopic,
		metrics:  &transport.NoOpMetrics{}, // no-op metrics by default
	}
}

// SetMetrics sets the metrics implementation.
func (rp *RetryProcessor) SetMetrics(metrics transport.Metrics) {
	rp.metrics = metrics
}

// ProcessWithRetry processes a message with retry logic.
func (rp *RetryProcessor) ProcessWithRetry(ctx context.Context, msg kafka.Message, handler transport.Handler) error {
	envelope, err := rp.parseMessage(msg)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse message")
		rp.metrics.IncMessagesProcessed(msg.Topic, "parse_error")
		return rp.sendToDLQ(ctx, msg, err, -1)
	}

	retryCount := rp.getRetryCount(msg)

	for attempt := 0; attempt <= rp.config.RetryCount; attempt++ {
		err = handler.Handle(ctx, *envelope)
		if err == nil {
			// Successful processing
			if attempt > 0 {
				log.Info().
					Str("event_id", envelope.EventID).
					Int("retry_count", attempt).
					Msg("Message processed successfully after retry")
				rp.metrics.IncMessagesProcessed(msg.Topic, "retry_success")
			}
			return nil
		}

		// Record retry attempt metric
		if attempt > 0 {
			rp.metrics.IncRetryAttempts(msg.Topic, attempt)
		}

		// Check whether we should retry
		if retryableErr, ok := err.(*RetryableError); ok && !retryableErr.Retryable {
			log.Error().
				Err(err).
				Str("event_id", envelope.EventID).
				Msg("Non-retryable error, sending to DLQ")
			rp.metrics.IncMessagesProcessed(msg.Topic, "non_retryable")
			return rp.sendToDLQ(ctx, msg, err, retryCount+attempt)
		}

		if attempt < rp.config.RetryCount {
			backoff := rp.config.GetRetryBackoffWithJitter(attempt)
			log.Warn().
				Err(err).
				Str("event_id", envelope.EventID).
				Int("attempt", attempt+1).
				Int("max_retries", rp.config.RetryCount).
				Dur("backoff", backoff).
				Msg("Retrying message processing")

			rp.metrics.IncMessagesProcessed(msg.Topic, "retry")

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				// Continue retrying
			}
		}
	}

	// All retry attempts exhausted
	log.Error().
		Err(err).
		Str("event_id", envelope.EventID).
		Int("total_retries", rp.config.RetryCount).
		Msg("All retry attempts exhausted, sending to DLQ")

	rp.metrics.IncMessagesProcessed(msg.Topic, "retry_exhausted")
	return rp.sendToDLQ(ctx, msg, err, retryCount+rp.config.RetryCount)
}

// parseMessage unmarshals a Kafka message into an Envelope.
func (rp *RetryProcessor) parseMessage(msg kafka.Message) (*transport.Envelope, error) {
	var envelope transport.Envelope
	if err := json.Unmarshal(msg.Value, &envelope); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}
	return &envelope, nil
}

// getRetryCount extracts the retry count from message headers.
func (rp *RetryProcessor) getRetryCount(msg kafka.Message) int {
	for _, header := range msg.Headers {
		if header.Key == rp.config.DLQRetryHeader {
			if count, err := strconv.Atoi(string(header.Value)); err == nil {
				return count
			}
		}
	}
	return 0
}

// sendToDLQ publishes the message to the configured Dead Letter Queue.
func (rp *RetryProcessor) sendToDLQ(ctx context.Context, originalMsg kafka.Message, processingErr error, totalRetries int) error {
	if !rp.config.DLQEnabled || rp.dlqTopic == "" {
		log.Warn().
			Str("original_topic", originalMsg.Topic).
			Msg("DLQ disabled, dropping message")
		return processingErr
	}

	// Build DLQ message with additional headers
	dlqMsg := kafka.Message{
		Topic:   rp.dlqTopic,
		Key:     originalMsg.Key,
		Value:   originalMsg.Value,
		Headers: rp.createDLQHeaders(originalMsg, processingErr, totalRetries),
	}

	// Use separate context so delivery to DLQ does not depend on the caller context
	publishCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := rp.producer.Publish(publishCtx, rp.dlqTopic, string(dlqMsg.Key), dlqMsg.Value); err != nil {
		log.Error().
			Err(err).
			Str("dlq_topic", rp.dlqTopic).
			Str("original_topic", originalMsg.Topic).
			Msg("Failed to send message to DLQ")
		return fmt.Errorf("failed to send to DLQ: %w", err)
	}

	// Record DLQ metric
	rp.metrics.IncDLQMessages(originalMsg.Topic, rp.dlqTopic)
	rp.metrics.IncMessagesProcessed(originalMsg.Topic, "dlq")

	log.Info().
		Str("dlq_topic", rp.dlqTopic).
		Str("original_topic", originalMsg.Topic).
		Int("partition", originalMsg.Partition).
		Int64("offset", originalMsg.Offset).
		Int("total_retries", totalRetries).
		Msg("Message sent to DLQ")

	return nil
}

// createDLQHeaders builds headers for a DLQ message.
func (rp *RetryProcessor) createDLQHeaders(originalMsg kafka.Message, err error, totalRetries int) []kafka.Header {
	headers := make([]kafka.Header, 0, len(originalMsg.Headers)+4)

	// Copy original headers
	for _, header := range originalMsg.Headers {
		// Skip retry headers to avoid duplicates
		if header.Key != rp.config.DLQRetryHeader {
			headers = append(headers, header)
		}
	}

	// Add DLQ specific headers
	headers = append(headers, kafka.Header{
		Key:   rp.config.DLQRetryHeader,
		Value: []byte(strconv.Itoa(totalRetries)),
	})

	headers = append(headers, kafka.Header{
		Key:   rp.config.DLQErrorHeader,
		Value: []byte(err.Error()),
	})

	headers = append(headers, kafka.Header{
		Key:   rp.config.DLQTimestampHeader,
		Value: []byte(time.Now().UTC().Format(time.RFC3339)),
	})

	// Include information about the original topic
	headers = append(headers, kafka.Header{
		Key:   "x-original-topic",
		Value: []byte(originalMsg.Topic),
	})

	headers = append(headers, kafka.Header{
		Key:   "x-original-partition",
		Value: []byte(strconv.Itoa(originalMsg.Partition)),
	})

	headers = append(headers, kafka.Header{
		Key:   "x-original-offset",
		Value: []byte(strconv.FormatInt(originalMsg.Offset, 10)),
	})

	return headers
}

// IsRetryableError determines whether an error should be retried.
func IsRetryableError(err error) bool {
	if retryableErr, ok := err.(*RetryableError); ok {
		return retryableErr.Retryable
	}

	// By default we treat errors as retryable except for specific cases.
	// Additional logic for non-retryable errors can be added here.
	switch err {
	case context.Canceled, context.DeadlineExceeded:
		return false
	default:
		return true
	}
}
