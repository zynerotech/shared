# App Bootstrap Package

Этот пакет предоставляет общую инфраструктуру для инициализации микросервисов.

## Основные компоненты

Пакет инициализирует следующие общие компоненты:
- Logger (логирование)
- Metrics (метрики)
- Healthcheck (проверка здоровья)
- Server (HTTP сервер)
- GRPC Server (gRPC сервер)
- Database (база данных)
- Cache (кэш)
- EventPublisher (издатель событий)

## Использование

### Базовое использование

```go
package main

import (
    "log"
    "gitlab.com/zynero/pim/products/internal/platform/config"
    app "gitlab.com/zynero/pim/products/pkg/app"
)

func main() {
    // Загружаем конфигурацию и инициализируем приложение
    cfg := &config.Config{}
    appBootstrap, err := app.BootstrapWithConfig(cfg, "")
    if err != nil {
        log.Fatalf("bootstrap failed: %v", err)
    }

    defer func() {
        if closeErr := appBootstrap.Close(); closeErr != nil {
            log.Printf("Error closing app bootstrap: %v", closeErr)
        }
    }()

    // Используем компоненты приложения
    logger := appBootstrap.Logger
    server := appBootstrap.Server
    // ...
}
```

### Расширенное использование

Если нужна более тонкая настройка процесса загрузки конфигурации:

```go
package main

import (
    "log"
    "gitlab.com/zynero/pim/products/internal/platform/config"
    app "gitlab.com/zynero/pim/products/pkg/app"
    platformconfig "gitlab.com/zynero/shared/config"
)

func main() {
    // Загружаем конфигурацию вручную
    cfg := &config.Config{}
    if err := platformconfig.Load(cfg, "configs/prod.yaml"); err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Инициализируем приложение
    appBootstrap, err := app.New(cfg)
    if err != nil {
        log.Fatalf("bootstrap failed: %v", err)
    }

    defer func() {
        if closeErr := appBootstrap.Close(); closeErr != nil {
            log.Printf("Error closing app bootstrap: %v", closeErr)
        }
    }()

    // Используем компоненты приложения
    // ...
}
```

## Интерфейс ConfigProvider

Для использования пакета ваша конфигурация должна реализовывать интерфейс `ConfigProvider`:

```go
type ConfigProvider interface {
    Validate() error
    LoggerConfig() platformlogger.Config
    MetricsConfig() platformmetrics.Config
    HealthcheckConfig() platformhealthcheck.Config
    ServerConfig() platformserver.Config
    DatabaseConfig() platformdatabase.Config
    CacheConfig() platformcache.Config
    KafkaConfig() kafka.Config
    GRPCConfig() platformgrpc.Config
}
```

## Функции

### BootstrapWithConfig

`BootstrapWithConfig(cfg ConfigProvider, configPath string) (*App, error)`

Объединяет загрузку конфигурации и инициализацию приложения в одном месте. Эта функция устраняет дублирование кода в точках входа.

**Параметры:**
- `cfg` - экземпляр конфигурации, реализующий интерфейс `ConfigProvider`
- `configPath` - путь к файлу конфигурации (пустая строка для использования по умолчанию)

**Возвращает:**
- `*App` - инициализированное приложение
- `error` - ошибка, если что-то пошло не так

### New

`New(cfg ConfigProvider) (*App, error)`

Инициализирует все общие инфраструктурные сервисы на основе предоставленной конфигурации.

**Параметры:**
- `cfg` - конфигурация, реализующая интерфейс `ConfigProvider`

**Возвращает:**
- `*App` - инициализированное приложение
- `error` - ошибка, если что-то пошло не так

### Close

`Close() error`

Останавливает метрики, проверки здоровья и закрывает соединения с базой данных.

**Возвращает:**
- `error` - ошибка, если что-то пошло не так при закрытии

