package kafka

import (
	"time"

	"github.com/segmentio/kafka-go"
)

// Config contains parameters for connecting to Kafka.
type Config struct {
	Brokers     []string          `mapstructure:"brokers" validate:"required,min=1"`
	SASL        *SASLConfig       `mapstructure:"sasl"`
	Producer    ProducerConfig    `mapstructure:"producer"`
	Consumer    ConsumerConfig    `mapstructure:"consumer"`
	Reliability ReliabilityConfig `mapstructure:"reliability"`
}

// SASLConfig describes SASL authentication settings.
type SASLConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Mechanism string `mapstructure:"mechanism" validate:"oneof=PLAIN SCRAM-SHA-256 SCRAM-SHA-512"`
	Username  string `mapstructure:"username"`
	Password  string `mapstructure:"password"`
}

// ProducerConfig holds producer related settings.
type ProducerConfig struct {
	// Default topic used when none is provided to Publish
	Topic        string        `mapstructure:"topic"`
	Compression  string        `mapstructure:"compression" validate:"oneof=none gzip snappy lz4 zstd"`
	BatchSize    int           `mapstructure:"batch_size" validate:"min=1,max=1000000"`
	BatchTimeout time.Duration `mapstructure:"batch_timeout" validate:"min=1ms"`
	RequiredAcks int           `mapstructure:"required_acks" validate:"oneof=-1 0 1"`
	MaxRetries   int           `mapstructure:"max_retries" validate:"min=0,max=10"`
	RetryBackoff time.Duration `mapstructure:"retry_backoff" validate:"min=1ms"`
}

// ConsumerConfig holds consumer related settings.
type ConsumerConfig struct {
	GroupID           string        `mapstructure:"group_id" validate:"required"`
	MinBytes          int           `mapstructure:"min_bytes" validate:"min=1"`
	MaxBytes          int           `mapstructure:"max_bytes" validate:"min=1"`
	MaxWait           time.Duration `mapstructure:"max_wait" validate:"min=1ms"`
	StartOffset       string        `mapstructure:"start_offset" validate:"oneof=earliest latest"`
	CommitInterval    time.Duration `mapstructure:"commit_interval" validate:"min=1ms"`
	MaxRetries        int           `mapstructure:"max_retries" validate:"min=0,max=10"`
	RetryBackoff      time.Duration `mapstructure:"retry_backoff" validate:"min=1ms"`
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval" validate:"min=1s"`
	SessionTimeout    time.Duration `mapstructure:"session_timeout" validate:"min=1s"`
	RebalanceTimeout  time.Duration `mapstructure:"rebalance_timeout" validate:"min=1s"`
}

// ReliabilityConfig configures retry and DLQ behaviour.
type ReliabilityConfig struct {
	// Retry options
	RetryCount             int           `mapstructure:"retry_count" validate:"min=0,max=10"`              // maximum number of attempts
	RetryBackoff           time.Duration `mapstructure:"retry_backoff" validate:"min=1ms"`                 // base delay between retries
	RetryBackoffMultiplier float64       `mapstructure:"retry_backoff_multiplier" validate:"min=1,max=10"` // multiplier for exponential backoff
	MaxRetryBackoff        time.Duration `mapstructure:"max_retry_backoff" validate:"min=1s"`              // upper limit for backoff

	// Dead Letter Queue options
	DLQTopic           string `mapstructure:"dlq_topic"`            // target topic for DLQ messages
	DLQEnabled         bool   `mapstructure:"dlq_enabled"`          // enable sending to DLQ
	DLQRetryHeader     string `mapstructure:"dlq_retry_header"`     // header storing retry count
	DLQErrorHeader     string `mapstructure:"dlq_error_header"`     // header storing error message
	DLQTimestampHeader string `mapstructure:"dlq_timestamp_header"` // header storing failure timestamp

	// Other options
	EnableMetrics        bool                 `mapstructure:"enable_metrics"`  // expose Prometheus metrics
	CircuitBreakerConfig CircuitBreakerConfig `mapstructure:"circuit_breaker"` // circuit breaker settings
}

// CircuitBreakerConfig contains settings for the circuit breaker.
type CircuitBreakerConfig struct {
	Enabled          bool          `mapstructure:"enabled"`
	FailureThreshold int           `mapstructure:"failure_threshold" validate:"min=1"`
	SuccessThreshold int           `mapstructure:"success_threshold" validate:"min=1"`
	Timeout          time.Duration `mapstructure:"timeout" validate:"min=1s"`
	MaxRequests      int           `mapstructure:"max_requests" validate:"min=1"`
}

// GetCompressionCodec converts the configured compression string to kafka.Compression.
func (pc *ProducerConfig) GetCompressionCodec() kafka.Compression {
	switch pc.Compression {
	case "gzip":
		return kafka.Gzip
	case "snappy":
		return kafka.Snappy
	case "lz4":
		return kafka.Lz4
	case "zstd":
		return kafka.Zstd
	case "none":
		return 0 // kafka.Compression(0) для "none"
	default:
		return kafka.Snappy // По умолчанию snappy
	}
}

// GetRetryBackoffWithJitter calculates retry delay with jitter applied.
func (rc *ReliabilityConfig) GetRetryBackoffWithJitter(attempt int) time.Duration {
	backoff := rc.RetryBackoff
	for i := 0; i < attempt; i++ {
		backoff = time.Duration(float64(backoff) * rc.RetryBackoffMultiplier)
		if backoff > rc.MaxRetryBackoff {
			backoff = rc.MaxRetryBackoff
			break
		}
	}
	return backoff
}

// GetDefaultReliabilityConfig returns default reliability settings.
func GetDefaultReliabilityConfig() ReliabilityConfig {
	return ReliabilityConfig{
		RetryCount:             3,
		RetryBackoff:           time.Second,
		RetryBackoffMultiplier: 2.0,
		MaxRetryBackoff:        30 * time.Second,
		DLQEnabled:             true,
		DLQRetryHeader:         "x-retry-count",
		DLQErrorHeader:         "x-error-message",
		DLQTimestampHeader:     "x-failed-timestamp",
		EnableMetrics:          false,
	}
}
