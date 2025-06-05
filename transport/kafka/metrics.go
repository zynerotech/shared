// Package kafka contains a Prometheus metrics implementation used by the Kafka
// transport. Metric names are derived from the provided service name. Labels for
// each metric are documented below:
//   - messages_received_total     {topic, partition}
//   - messages_processed_total    {topic, status}
//   - message_processing_duration_seconds {topic}
//   - retry_attempts_total        {topic, attempt}
//   - messages_sent_total         {topic, status}
//   - message_publish_duration_seconds {topic}
//   - dlq_messages_total          {original_topic, dlq_topic}
//   - active_consumers            no labels
//   - active_producers            no labels
//   - uptime_seconds              no labels
package kafka

import (
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// KafkaMetrics provides a Prometheus metrics implementation used by the Kafka
// transport and integrates with the shared metrics package.
type KafkaMetrics struct {
	// Consumer metrics
	messagesReceived  *prometheus.CounterVec
	messagesProcessed *prometheus.CounterVec
	processingTime    *prometheus.HistogramVec
	retryAttempts     *prometheus.CounterVec

	// Producer metrics
	messagesSent *prometheus.CounterVec
	publishTime  *prometheus.HistogramVec

	// DLQ metrics
	dlqMessages *prometheus.CounterVec

	// Common metrics
	activeConsumers prometheus.Gauge
	activeProducers prometheus.Gauge
	uptime          prometheus.Gauge

	startTime time.Time
	mu        sync.RWMutex
	stopCh    chan struct{}
	doneCh    chan struct{}
}

// NewKafkaMetrics creates a new metrics collector for the Kafka transport.
func NewKafkaMetrics(serviceName string) *KafkaMetrics {
	if serviceName == "" {
		serviceName = "kafka_transport"
	}

	m := &KafkaMetrics{
		startTime: time.Now(),
		stopCh:    make(chan struct{}),
		doneCh:    make(chan struct{}),
	}

	// Consumer metrics
	m.messagesReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_messages_received_total", serviceName),
			Help: "Total number of messages received from Kafka topics",
		},
		[]string{"topic", "partition"},
	)

	m.messagesProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_messages_processed_total", serviceName),
			Help: "Total number of messages processed",
		},
		// status label has values: success, error, retry, dlq
		[]string{"topic", "status"},
	)

	m.processingTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    fmt.Sprintf("%s_message_processing_duration_seconds", serviceName),
			Help:    "Time spent processing messages",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"topic"},
	)

	m.retryAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_retry_attempts_total", serviceName),
			Help: "Total number of retry attempts",
		},
		[]string{"topic", "attempt"},
	)

	// Producer metrics
	m.messagesSent = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_messages_sent_total", serviceName),
			Help: "Total number of messages sent to Kafka topics",
		},
		// status label has values: success, error
		[]string{"topic", "status"},
	)

	m.publishTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    fmt.Sprintf("%s_message_publish_duration_seconds", serviceName),
			Help:    "Time spent publishing messages",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"topic"},
	)

	// DLQ metrics
	m.dlqMessages = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_dlq_messages_total", serviceName),
			Help: "Total number of messages sent to Dead Letter Queue",
		},
		[]string{"original_topic", "dlq_topic"},
	)

	// Common metrics
	m.activeConsumers = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: fmt.Sprintf("%s_active_consumers", serviceName),
			Help: "Number of active consumers",
		},
	)

	m.activeProducers = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: fmt.Sprintf("%s_active_producers", serviceName),
			Help: "Number of active producers",
		},
	)

	m.uptime = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: fmt.Sprintf("%s_uptime_seconds", serviceName),
			Help: "Service uptime in seconds",
		},
	)

	// Start background goroutine that updates the uptime metric.
	go m.updateUptimeLoop()

	return m
}

// Consumer metrics
func (m *KafkaMetrics) IncMessagesReceived(topic string, partition int) {
	m.messagesReceived.WithLabelValues(topic, fmt.Sprintf("%d", partition)).Inc()
}

func (m *KafkaMetrics) IncMessagesProcessed(topic string, status string) {
	m.messagesProcessed.WithLabelValues(topic, status).Inc()
}

func (m *KafkaMetrics) RecordProcessingTime(topic string, duration time.Duration) {
	m.processingTime.WithLabelValues(topic).Observe(duration.Seconds())
}

func (m *KafkaMetrics) IncRetryAttempts(topic string, attempt int) {
	m.retryAttempts.WithLabelValues(topic, fmt.Sprintf("%d", attempt)).Inc()
}

// Producer metrics
func (m *KafkaMetrics) IncMessagesSent(topic string, status string) {
	m.messagesSent.WithLabelValues(topic, status).Inc()
}

func (m *KafkaMetrics) RecordPublishTime(topic string, duration time.Duration) {
	m.publishTime.WithLabelValues(topic).Observe(duration.Seconds())
}

// DLQ metrics
func (m *KafkaMetrics) IncDLQMessages(originalTopic, dlqTopic string) {
	m.dlqMessages.WithLabelValues(originalTopic, dlqTopic).Inc()
}

// Common metrics
func (m *KafkaMetrics) SetActiveConsumers(count int) {
	m.activeConsumers.Set(float64(count))
}

func (m *KafkaMetrics) SetActiveProducers(count int) {
	m.activeProducers.Set(float64(count))
}

func (m *KafkaMetrics) RecordUptime(duration time.Duration) {
	m.uptime.Set(duration.Seconds())
}

// updateUptimeLoop updates the uptime metric every 10 seconds until Close is
// called.
func (m *KafkaMetrics) updateUptimeLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			uptime := time.Since(m.startTime)
			m.RecordUptime(uptime)
		case <-m.stopCh:
			close(m.doneCh)
			return
		}
	}
}

// Close stops internal goroutines and releases resources.
func (m *KafkaMetrics) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	select {
	case <-m.stopCh:
		// already closed
		return
	default:
		close(m.stopCh)
	}

	<-m.doneCh
}
