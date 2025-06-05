package transport

import (
	"time"
)

// Metrics определяет интерфейс для сбора метрик транспорта
type Metrics interface {
	// Consumer метрики
	IncMessagesReceived(topic string, partition int)
	IncMessagesProcessed(topic string, status string) // status: success, error, retry, dlq
	RecordProcessingTime(topic string, duration time.Duration)
	IncRetryAttempts(topic string, attempt int)

	// Producer метрики
	IncMessagesSent(topic string, status string) // status: success, error
	RecordPublishTime(topic string, duration time.Duration)

	// DLQ метрики
	IncDLQMessages(originalTopic, dlqTopic string)

	// Общие метрики
	SetActiveConsumers(count int)
	SetActiveProducers(count int)
	RecordUptime(duration time.Duration)
}

// NoOpMetrics реализация метрик, которая ничего не делает (для тестов/отключения)
type NoOpMetrics struct{}

func (m *NoOpMetrics) IncMessagesReceived(topic string, partition int)           {}
func (m *NoOpMetrics) IncMessagesProcessed(topic string, status string)          {}
func (m *NoOpMetrics) RecordProcessingTime(topic string, duration time.Duration) {}
func (m *NoOpMetrics) IncRetryAttempts(topic string, attempt int)                {}
func (m *NoOpMetrics) IncMessagesSent(topic string, status string)               {}
func (m *NoOpMetrics) RecordPublishTime(topic string, duration time.Duration)    {}
func (m *NoOpMetrics) IncDLQMessages(originalTopic, dlqTopic string)             {}
func (m *NoOpMetrics) SetActiveConsumers(count int)                              {}
func (m *NoOpMetrics) SetActiveProducers(count int)                              {}
func (m *NoOpMetrics) RecordUptime(duration time.Duration)                       {}
