# App Bootstrap Package

Пакет `app` предоставляет унифицированный и гибкий способ инициализации общих компонентов инфраструктуры для микросервисов. Теперь поддерживает **опциональные компоненты** - по умолчанию инициализируется только логирование, а остальные компоненты подключаются по необходимости.

## 🚀 Новые возможности

### Гибкая архитектура с опциональными компонентами
- **Минимальная инициализация** - только логгер по умолчанию
- **Выборочное подключение** компонентов через Builder паттерн
- **Обратная совместимость** с существующими конфигурациями
- **Типобезопасность** - проверка наличия компонентов на этапе компиляции

### Паттерн Builder
- **Fluent API** для удобного конфигурирования
- **Цепочка методов** для выборочного подключения компонентов
- **Обработка ошибок** на этапе сборки

## Основные компоненты

Пакет может инициализировать следующие компоненты (все опциональные, кроме Logger):
- **Logger** (логирование) - **обязательный компонент**
- Metrics (метрики) - опциональный
- Healthcheck (проверка здоровья) - опциональный
- Server (HTTP сервер) - опциональный
- GRPC Server (gRPC сервер) - опциональный
- Database (база данных) - опциональный
- Cache (кэш) - опциональный
- EventPublisher (издатель событий) - опциональный

## 📋 Интерфейсы конфигурации

### ConfigProvider (обязательный)
```go
type ConfigProvider interface {
    Validate() error
    LoggerConfig() platformlogger.Config
}
```

### OptionalConfigProvider (опциональный)
```go
type OptionalConfigProvider interface {
    MetricsConfig() *platformmetrics.Config
    HealthcheckConfig() *platformhealthcheck.Config
    ServerConfig() *platformserver.Config
    DatabaseConfig() *platformdatabase.Config
    CacheConfig() *platformcache.Config
    KafkaConfig() *kafka.Config
    GRPCConfig() *platformgrpc.Config
}
```

**Важно**: Методы `OptionalConfigProvider` должны возвращать `nil`, если компонент не нужен.

## 🎯 Способы использования

### 1. Минимальная инициализация (только логгер)

```go
type Config struct {
    Logger platformlogger.Config `mapstructure:"logger"`
}

func (c Config) Validate() error { return nil }
func (c Config) LoggerConfig() platformlogger.Config { return c.Logger }

// Инициализация
app, err := app.NewWithLogger(cfg)
if err != nil {
    log.Fatal(err)
}
defer app.Close()
```

### 2. Выборочное подключение компонентов

```go
type Config struct {
    Logger    platformlogger.Config `mapstructure:"logger"`
    Metrics   *platformmetrics.Config `mapstructure:"metrics"`
    Database  *platformdatabase.Config `mapstructure:"database"`
}

func (c Config) Validate() error { return nil }
func (c Config) LoggerConfig() platformlogger.Config { return c.Logger }
func (c Config) MetricsConfig() *platformmetrics.Config { return c.Metrics }
func (c Config) DatabaseConfig() *platformdatabase.Config { return c.Database }

// Инициализация через Builder
app, err := app.NewBuilder(cfg).
    WithLogger().
    WithMetrics().
    WithDatabase().
    Build()
if err != nil {
    log.Fatal(err)
}
defer app.Close()
```

### 3. Все компоненты (legacy поведение)

```go
type Config struct {
    Logger      platformlogger.Config `mapstructure:"logger"`
    Metrics     platformmetrics.Config `mapstructure:"metrics"`
    Healthcheck platformhealthcheck.Config `mapstructure:"healthcheck"`
    Server      platformserver.Config `mapstructure:"server"`
    Database    platformdatabase.Config `mapstructure:"database"`
    Cache       platformcache.Config `mapstructure:"cache"`
    Kafka       kafka.Config `mapstructure:"kafka"`
    GRPC        platformgrpc.Config `mapstructure:"grpc"`
}

func (c Config) Validate() error { return nil }
func (c Config) LoggerConfig() platformlogger.Config { return c.Logger }
func (c Config) MetricsConfig() *platformmetrics.Config { return &c.Metrics }
func (c Config) HealthcheckConfig() *platformhealthcheck.Config { return &c.Healthcheck }
func (c Config) ServerConfig() *platformserver.Config { return &c.Server }
func (c Config) DatabaseConfig() *platformdatabase.Config { return &c.Database }
func (c Config) CacheConfig() *platformcache.Config { return &c.Cache }
func (c Config) KafkaConfig() *kafka.Config { return &c.Kafka }
func (c Config) GRPCConfig() *platformgrpc.Config { return &c.GRPC }

// Инициализация всех компонентов
app, err := app.New(cfg)
if err != nil {
    log.Fatal(err)
}
defer app.Close()
```

### 4. Загрузка конфигурации из файла

```go
// Минимальная инициализация с загрузкой конфигурации
app, err := app.BootstrapWithConfigAndLogger(cfg, "config.yaml")
if err != nil {
    log.Fatal(err)
}
defer app.Close()

// Полная инициализация с загрузкой конфигурации
app, err := app.BootstrapWithConfig(cfg, "config.yaml")
if err != nil {
    log.Fatal(err)
}
defer app.Close()
```

## 🔧 Builder API

### Доступные методы
- `WithLogger()` - инициализирует логгер (обязательный)
- `WithMetrics()` - инициализирует метрики (если конфигурация предоставлена)
- `WithHealthcheck()` - инициализирует healthcheck (если конфигурация предоставлена)
- `WithServer()` - инициализирует HTTP сервер (если конфигурация предоставлена)
- `WithDatabase()` - инициализирует базу данных (если конфигурация предоставлена)
- `WithCache()` - инициализирует кэш (если конфигурация предоставлена)
- `WithKafka()` - инициализирует Kafka producer (если конфигурация предоставлена)
- `WithGRPC()` - инициализирует gRPC сервер (если конфигурация предоставлена)
- `WithAll()` - инициализирует все доступные компоненты
- `Build()` - создает экземпляр App

### Пример цепочки методов
```go
app, err := app.NewBuilder(cfg).
    WithLogger().
    WithMetrics().
    WithServer().
    WithDatabase().
    Build()
```

## 🛡️ Безопасность компонентов

Все компоненты в структуре `App` могут быть `nil`. Всегда проверяйте их наличие перед использованием:

```go
if app.Metrics != nil {
    // Использование метрик
}

if app.Database != nil {
    // Работа с базой данных
}

if app.Server != nil {
    // Запуск HTTP сервера
}
```

## 📝 Пример конфигурации YAML

```yaml
# Минимальная конфигурация (только логгер)
logger:
  level: info
  format: json
  output: stdout

# Опциональные компоненты (подключаются только если нужны)
metrics:
  enabled: true
  path: /metrics
  port: 9090

database:
  host: localhost
  port: 5432
  user: postgres
  password: password
  dbname: myapp

server:
  address: :8080
  read_timeout: 30s
  write_timeout: 30s
```

## 🔄 Миграция с предыдущих версий

Для существующих проектов достаточно изменить возвращаемые типы в методах конфигурации:

```go
// Было
func (c Config) MetricsConfig() platformmetrics.Config { return c.Metrics }

// Стало
func (c Config) MetricsConfig() *platformmetrics.Config { return &c.Metrics }
```

Или использовать новый Builder API для более гибкого контроля.

