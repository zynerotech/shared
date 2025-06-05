# Работа с Apache Kafka

Пакет помогает работать с Apache Kafka, содержит интерфейсы и реализацию для консьюмеров и продьюсеров с поддержкой graceful shutdown, retry, Dead Letter Queue (DLQ) и метрик.

## Структура файлов

### Базовый пакет (`transport/`)
* `consumer.go` - интерфейс для консьюмера с поддержкой graceful shutdown
* `event_publisher.go` - интерфейс для публикатора событий
* `handler.go` - интерфейс для обработчиков входящих сообщений
* `retry.go` - утилиты и интерфейсы для retry механизмов
* `metrics.go` - интерфейс для метрик транспорта
* `message.go` - структура сообщения
* `producer.go` - интерфейс продьюсера сообщений в транспорт с поддержкой закрытия

### Kafka реализация (`transport/kafka/`)
* `consumer.go` - реализация консьюмера с graceful shutdown, retry/DLQ и метриками
* `producer.go` - реализация продьюсера с graceful shutdown и метриками
* `retry.go` - реализация retry и DLQ механизмов для Kafka с метриками
* `metrics.go` - реализация метрик для Kafka транспорта (интеграция с `@/metrics`)
* `config.go` - расширенная конфигурация с настройками retry и DLQ
* `event_publisher.go` - реализация интерфейса публикатора событий для Kafka

### Примеры и документация
* `cmd/example/main.go` - пример использования с graceful shutdown, retry, DLQ и observability

## Особенности

### Graceful Shutdown
- **Consumer**: поддерживает корректное завершение с методами `Stop()`, `Wait()`, `Close()`
- **Producer**: поддерживает метод `Close()` для ожидания отправки буферизованных сообщений
- **Контекст**: все операции поддерживают отмену через context
- **Таймауты**: настраиваемые таймауты для shutdown операций

### Retry Механизмы
- **Экспоненциальный backoff**: настраиваемая задержка между попытками
- **Максимальное количество retry**: предотвращение бесконечных попыток
- **Типизированные ошибки**: различие между временными и постоянными ошибками
- **Jitter**: добавление случайности для избежания thundering herd

### Dead Letter Queue (DLQ)
- **Автоматическая отправка**: сообщения, которые не удалось обработать, попадают в DLQ
- **Метаданные**: сохранение информации об ошибках и количестве retry
- **Заголовки**: детальная информация о причинах попадания в DLQ
- **Мониторинг**: возможность обработки DLQ сообщений отдельным consumer

### Observability

#### Интеграция с существующими пакетами
- **Metrics**: использует `@/metrics` пакет для Prometheus метрик
- **Health checks**: использует `@/healthcheck` пакет для проверок здоровья

#### Метрики Kafka транспорта
- **Consumer метрики**:
  - `{service}_messages_received_total` - количество полученных сообщений
  - `{service}_messages_processed_total` - количество обработанных сообщений (по статусам)
  - `{service}_message_processing_duration_seconds` - время обработки сообщений
  - `{service}_retry_attempts_total` - количество retry попыток
  
- **Producer метрики**:
  - `{service}_messages_sent_total` - количество отправленных сообщений
  - `{service}_message_publish_duration_seconds` - время публикации сообщений
  
- **DLQ метрики**:
  - `{service}_dlq_messages_total` - количество сообщений в DLQ
  
- **Общие метрики**:
  - `{service}_active_consumers` - количество активных consumer
  - `{service}_active_producers` - количество активных producer
  - `{service}_uptime_seconds` - время работы сервиса

### Надежность
- Ручное управление коммитами в Consumer
- Обработка ошибок без panic
- Структурированное логирование
- Circuit breaker (в конфигурации)

## Конфигурация

### Retry настройки
```go
Reliability: kafka.ReliabilityConfig{
    RetryCount:             3,                // Максимальное количество retry
    RetryBackoff:           time.Second,      // Базовая задержка
    RetryBackoffMultiplier: 2.0,             // Множитель для экспоненциального backoff
    MaxRetryBackoff:        30 * time.Second, // Максимальная задержка
}
```

### DLQ настройки
```go
Reliability: kafka.ReliabilityConfig{
    DLQEnabled:           true,                // Включить DLQ
    DLQTopic:             "my-topic-dlq",     // Топик для DLQ
    DLQRetryHeader:       "x-retry-count",    // Заголовок с количеством retry
    DLQErrorHeader:       "x-error-message",  // Заголовок с текстом ошибки
    DLQTimestampHeader:   "x-failed-timestamp", // Заголовок с временем ошибки
}
```

### Observability настройки
```go
// Инициализация metrics сервера
metricsConfig := metrics.Config{
    Enabled:     true,
    Path:        "/metrics",
    Port:        8080,
    ServiceName: "my_service",
}

metricsManager, _ := metrics.New(metricsConfig)

// Инициализация healthcheck сервера
healthConfig := healthcheck.Config{
    Enabled: true,
    Path:    "/health",
    Port:    8081,
}

healthManager, _ := healthcheck.New(healthConfig)

// Создание Kafka метрик
kafkaMetrics := kafka.NewKafkaMetrics("my_service")

// Настройка компонентов
consumer.SetMetrics(kafkaMetrics)
producer.SetMetrics(kafkaMetrics)
```

## Типы ошибок

### Повторяемые ошибки
```go
// Обычная ошибка (повторяется по умолчанию)
return errors.New("temporary database connection error")

// Временная ошибка с указанием задержки
return transport.NewTemporaryError(err, 5*time.Second)
```

### Неповторяемые ошибки
```go
// Ошибка, которая попадает сразу в DLQ
return transport.NewNonRetryableError(errors.New("invalid message format"))
```

## Пример использования

### Базовая настройка с observability
```go
// Инициализация metrics и healthcheck серверов
metricsManager, _ := metrics.New(metrics.Config{
    Enabled: true, Path: "/metrics", Port: 8080, ServiceName: "example_service",
})
healthManager, _ := healthcheck.New(healthcheck.Config{
    Enabled: true, Path: "/health", Port: 8081,
})

// Создание Kafka метрик
kafkaMetrics := kafka.NewKafkaMetrics("example_service")

// Конфигурация с retry и DLQ
cfg := kafka.Config{
    Brokers: []string{"localhost:9092"},
    Reliability: kafka.ReliabilityConfig{
        RetryCount:             3,
        RetryBackoff:           time.Second,
        RetryBackoffMultiplier: 2.0,
        MaxRetryBackoff:        30 * time.Second,
        DLQEnabled:             true,
        DLQTopic:               "my-topic-dlq",
        EnableMetrics:          true,
    },
}

// Создание компонентов с метриками
consumer := kafka.NewConsumer(cfg, "my-topic", handler)
consumer.SetMetrics(kafkaMetrics)

producer, _ := kafka.NewProducer(cfg)
producer.SetMetrics(kafkaMetrics)
```

### Обработчик с различными типами ошибок
```go
func (h *Handler) Handle(ctx context.Context, envelope transport.Envelope) error {
    // Временная ошибка - будет retry
    if isTemporaryError() {
        return errors.New("temporary error")
    }
    
    // Постоянная ошибка - прямо в DLQ
    if isPermanentError() {
        return transport.NewNonRetryableError(errors.New("permanent error"))
    }
    
    // Успешная обработка
    return nil
}
```

### DLQ Consumer
```go
// Отдельный consumer для обработки DLQ сообщений
dlqConsumer := kafka.NewConsumer(cfg, "my-topic-dlq", dlqHandler)
dlqConsumer.SetMetrics(kafkaMetrics)

// DLQ обработчик для manual intervention
func (h *DLQHandler) Handle(ctx context.Context, envelope transport.Envelope) error {
    // Логирование, алерты, сохранение для анализа
    log.Error().Str("event_id", envelope.EventID).Msg("DLQ message requires attention")
    return nil
}
```

Подробный пример см. в `cmd/example/main.go`

## Мониторинг и алерты

### Prometheus запросы

**Количество ошибок**:
```promql
rate(example_service_messages_processed_total{status="error"}[5m])
```

**Время обработки P95**:
```promql
histogram_quantile(0.95, rate(example_service_message_processing_duration_seconds_bucket[5m]))
```

**Количество сообщений в DLQ**:
```promql
rate(example_service_dlq_messages_total[5m])
```

### Алерты Grafana/AlertManager

```yaml
groups:
- name: kafka-transport
  rules:
  - alert: HighErrorRate
    expr: rate(example_service_messages_processed_total{status="error"}[5m]) > 0.1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High error rate in Kafka transport"

  - alert: DLQMessages
    expr: rate(example_service_dlq_messages_total[5m]) > 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "Messages being sent to DLQ"
```

## Best Practices

1. **Настройка retry**: используйте разумные значения для retry count и backoff
2. **DLQ мониторинг**: настройте алерты на появление сообщений в DLQ
3. **Типизация ошибок**: четко разделяйте временные и постоянные ошибки
4. **Graceful shutdown**: всегда используйте корректное завершение работы
5. **Логирование**: используйте структурированные логи для анализа проблем
6. **Метрики**: настройте дашборды для мониторинга производительности
7. **Health checks**: используйте отдельный сервер для health checks
8. **Алерты**: настройте критичные алерты для быстрого реагирования
