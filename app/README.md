# App Bootstrap Package

Пакет `app` предоставляет унифицированный способ инициализации всех общих компонентов инфраструктуры для микросервисов, включая **расширенную интеграцию с глобальным логгером**.

## 🚀 Новые возможности

### Глобальная конфигурация логгера
- **Автоматическая инициализация** глобального логгера с метаданными приложения
- **Компонентное логирование** с индивидуальными настройками для каждого компонента
- **Динамическое управление** уровнями логирования и глобальными полями
- **Обратная совместимость** с существующими конфигурациями

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

## 📋 Интерфейс ConfigProvider

```go
type ConfigProvider interface {
    Validate() error
    LoggerConfig() platformlogger.Config
    GlobalLoggerConfig() *platformlogger.GlobalConfig
    MetricsConfig() platformmetrics.Config
    HealthcheckConfig() platformhealthcheck.Config
    ServerConfig() platformserver.Config
    DatabaseConfig() platformdatabase.Config
    CacheConfig() platformcache.Config
    KafkaConfig() kafka.Config
    GRPCConfig() platformgrpc.Config
}
```

## 🎯 Способы использования

### 1. Автоматическая глобальная конфигурация (рекомендуется)

```go
package main

import (
    "gitlab.com/zynero/shared/app"
    "gitlab.com/zynero/shared/logger"
)

type AppConfig struct {
    Logger logger.Config `mapstructure:"logger"`
    // ... другие конфигурации
}

func (c *AppConfig) Validate() error { return nil }
func (c *AppConfig) LoggerConfig() logger.Config { return c.Logger }
func (c *AppConfig) GlobalLoggerConfig() *logger.GlobalConfig { return nil } // Автоматическая генерация

func main() {
    cfg := &AppConfig{
        Logger: logger.Config{
            Level:      "debug",
            Format:     "console",
            Output:     "stdout",
            CallerInfo: true,
        },
    }

    // 🌟 Автоматически создает глобальную конфигурацию
    application, err := app.BootstrapWithGlobalConfig(cfg, "config.yaml", "user-service", "1.0.0")
    if err != nil {
        log.Fatal(err)
    }
    defer application.Close()

    // Теперь во всех пакетах можно использовать глобальный логгер
    logger.Info().Msg("Application started")
    logger.Component("database").Info().Msg("Database connected")
}
```

### 2. Предустановленная глобальная конфигурация

```go
type AppConfig struct {
    Logger    logger.Config         `mapstructure:"logger"`
    GlobalLog *logger.GlobalConfig  `mapstructure:"global_logger"`
}

func (c *AppConfig) GlobalLoggerConfig() *logger.GlobalConfig {
    return c.GlobalLog
}

func main() {
    cfg := &AppConfig{
        Logger: logger.Config{Level: "info", Format: "json"},
        GlobalLog: &logger.GlobalConfig{
            Logger: logger.Config{
                Level:      "debug",
                Format:     "console",
                CallerInfo: true,
            },
            Application: logger.ApplicationInfo{
                Name:        "user-service",
                Version:     "1.0.0",
                Environment: "production",
                Instance:    "server-01",
            },
            GlobalFields: map[string]any{
                "service_type": "microservice",
                "region":      "us-east-1",
            },
            Components: map[string]logger.ComponentConfig{
                "database": {
                    Level: "warn",
                    Fields: map[string]any{"db_type": "postgres"},
                },
                "api": {
                    Level: "debug",
                    Fields: map[string]any{"api_version": "v1"},
                },
            },
        },
    }

    application, err := app.BootstrapWithConfig(cfg, "config.yaml")
    if err != nil {
        log.Fatal(err)
    }
    defer application.Close()
}
```

### 3. Обратная совместимость

```go
// Старый способ продолжает работать
func (c *AppConfig) GlobalLoggerConfig() *logger.GlobalConfig {
    return nil // Используется старый способ инициализации
}

application, err := app.BootstrapWithConfig(cfg, "config.yaml")
```

## 🔧 Автоматическая глобальная конфигурация

При использовании `BootstrapWithGlobalConfig` автоматически создается:

```go
globalCfg := logger.GlobalConfig{
    Logger: cfg.LoggerConfig(),
    Application: logger.ApplicationInfo{
        Name:        appName,        // переданный параметр
        Version:     appVersion,     // переданный параметр
        Environment: getEnvironment(), // из ENV переменных
        Instance:    hostname,       // автоматически определяется
    },
    GlobalFields: map[string]any{
        "service_type": "microservice",
        "startup_time": time.Now().Format(time.RFC3339),
    },
    Components: map[string]logger.ComponentConfig{
        "app":      {Level: "info"},
        "database": {Level: "warn", Fields: map[string]any{"component_type": "database"}},
        "cache":    {Level: "info", Fields: map[string]any{"component_type": "cache"}},
        "kafka":    {Level: "info", Fields: map[string]any{"component_type": "message_broker"}},
        "grpc":     {Level: "info", Fields: map[string]any{"component_type": "grpc_server"}},
        "http":     {Level: "info", Fields: map[string]any{"component_type": "http_server"}},
    },
}
```

## 📊 Логирование в bootstrap

Обновленный bootstrap автоматически логирует все этапы инициализации:

```
2025-06-25T01:20:57+03:00 INF > Initializing application components 
  app_name=demo-service app_version=1.0.0 component=app environment=development

2025-06-25T01:20:57+03:00 WRN > Database connection established 
  app_name=demo-service component=database component_type=database

2025-06-25T01:20:57+03:00 INF > All components initialized successfully 
  app_name=demo-service environment=development
```

## 🎛️ Динамическое управление

После инициализации можно динамически управлять логгером:

```go
// Обновление глобальных полей
logger.UpdateGlobalFields(map[string]any{
    "request_id": "req-12345",
    "user_id":    "user-67890",
})

// Изменение уровня компонента
logger.SetComponentLevel("database", "debug")

// Просмотр информации
components := logger.ListComponents()
level := logger.GetComponentLevel("database")
config := logger.GetGlobalConfig()
```

## 🔍 Определение окружения

Автоматически определяется из переменных окружения:
1. `ENVIRONMENT` 
2. `ENV`
3. По умолчанию: `"development"`

## 📁 Структура проекта

```
app/
├── bootstrap.go           # Основной файл с обновленной логикой
├── example/              # Примеры использования
│   ├── main.go           # Полный пример с интерфейсами
│   ├── simple_example.go # Упрощенный пример
│   └── go.mod
└── README.md             # Эта документация
```

## 🚀 Преимущества

### ✅ Централизованное управление
- **Одна точка конфигурации** в bootstrap
- **Автоматическое распространение** на все компоненты
- **Единообразие** логирования во всем приложении

### ✅ Контекстная информация
- **Метаданные приложения** во всех сообщениях
- **Компонентная трассировка** с автоматическими полями
- **Инфраструктурная информация** (регион, кластер, инстанс)

### ✅ Гранулярное управление
- **Разные уровни** для разных компонентов
- **Специфические поля** для каждого компонента
- **Динамическое изменение** в рантайме

### ✅ Обратная совместимость
- **Существующий код** продолжает работать
- **Постепенная миграция** на новые возможности
- **Гибкость** в выборе способа конфигурации

## 🔄 Миграция

### С существующего кода:
1. **Добавьте метод** `GlobalLoggerConfig()` в ваш `ConfigProvider`
2. **Замените вызов** `BootstrapWithConfig` на `BootstrapWithGlobalConfig`
3. **Используйте компонентные логгеры** в ваших пакетах

### Пример миграции:
```go
// Было
func (c *Config) GlobalLoggerConfig() *logger.GlobalConfig {
    return nil // старый способ
}
app.BootstrapWithConfig(cfg, "config.yaml")

// Стало
func (c *Config) GlobalLoggerConfig() *logger.GlobalConfig {
    return nil // автоматическая генерация
}
app.BootstrapWithGlobalConfig(cfg, "config.yaml", "my-service", "1.0.0")
```

## 🎯 Лучшие практики

1. **Используйте `BootstrapWithGlobalConfig`** для новых приложений
2. **Определяйте компоненты** в конфигурации для лучшей организации
3. **Используйте компонентные логгеры** в соответствующих пакетах
4. **Добавляйте контекстные поля** для трассировки
5. **Настраивайте уровни** в зависимости от окружения

