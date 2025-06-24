# Logger Package

Пакет `logger` представляет собой удобную обертку над [zerolog](https://github.com/rs/zerolog), предоставляющую унифицированный интерфейс для структурированного логирования. Этот пакет можно использовать в сторонних приложениях как полноценную замену прямого использования zerolog.

## Возможности

- **Простой API**: Удобные глобальные функции для быстрого логирования
- **Структурированное логирование**: Поддержка полей различных типов
- **Контекстное логирование**: Создание логгеров с предустановленными полями
- **Гибкая конфигурация**: JSON/Console форматы, различные уровни логирования
- **Производительность**: Основан на быстром zerolog
- **Go context поддержка**: Интеграция с стандартным пакетом context
- **Обратная совместимость**: Доступ к базовому zerolog.Logger при необходимости

## Установка

```bash
go get gitlab.com/zynero/shared/logger
```

## Быстрый старт

### Базовое использование

```go
package main

import "gitlab.com/zynero/shared/logger"

func main() {
    // Простое логирование
    logger.Info().Msg("Application started")
    logger.Error().Msg("Something went wrong")
    
    // Форматированный вывод
    logger.Infof("User %s logged in", "john_doe")
}
```

### Конфигурация логгера

```go
package main

import (
    "gitlab.com/zynero/shared/logger"
    "time"
)

func main() {
    cfg := logger.Config{
        Level:      "debug",           // trace, debug, info, warn, error, fatal, panic
        Format:     "console",         // json или console
        Output:     "stdout",          // stdout, stderr или путь к файлу
        TimeFormat: time.RFC3339,      // формат времени
        CallerInfo: true,              // добавлять информацию о вызывающем коде
    }
    
    if err := logger.Init(cfg); err != nil {
        panic(err)
    }
    
    logger.Info().Msg("Logger configured successfully")
}
```

## Основное использование

### Уровни логирования

```go
logger.Trace().Msg("Trace level message")
logger.Debug().Msg("Debug level message")
logger.Info().Msg("Info level message")
logger.Warn().Msg("Warning message")
logger.Error().Msg("Error message")
logger.Fatal().Msg("Fatal message") // завершает программу
logger.Panic().Msg("Panic message") // вызывает панику
```

### Форматированное логирование

```go
logger.Infof("Processing %d items for user %s", 42, "john_doe")
logger.Errorf("Failed to connect to database: %v", err)
```

### Логирование с полями

```go
// Добавление полей к событию
logger.Info().
    Str("user_id", "12345").
    Int("items_count", 10).
    Bool("is_premium", true).
    Dur("response_time", 150*time.Millisecond).
    Msg("User action completed")

// Работа с ошибками
err := errors.New("database connection failed")
logger.Error().Err(err).Msg("Operation failed")
```

### Контекстные логгеры

```go
// Логгер с одним полем
userLogger := logger.WithField("user_id", "12345")
userLogger.Info().Msg("User-specific operation")

// Логгер с несколькими полями
serviceLogger := logger.WithFields(map[string]any{
    "service":    "user-service",
    "version":    "1.2.3",
    "request_id": "req-123",
})
serviceLogger.Info().Msg("Service operation completed")

// Логгер с ошибкой
errorLogger := logger.WithError(err)
errorLogger.Info().Msg("Continuing with error context")
```

### Построитель логгера

```go
// Создание логгера с множеством полей
customLogger := logger.With().
    Str("component", "database").
    Int64("connection_id", 12345).
    Float64("cpu_usage", 23.5).
    Time("timestamp", time.Now()).
    Interface("metadata", map[string]string{"region": "us-east-1"}).
    Logger()

customLogger.Info().Msg("Database operation completed")
```

### Работа с Go context

```go
ctx := context.Background()
ctxLogger := logger.WithContext(ctx)
ctxLogger.Info().Msg("Operation with context")
```

## Создание отдельных экземпляров

```go
// Логгер для записи в файл
fileCfg := logger.Config{
    Level:  "error",
    Format: "json",
    Output: "error.log",
}

fileLogger, err := logger.New(fileCfg)
if err != nil {
    log.Fatal(err)
}

fileLogger.Error().Str("component", "database").Msg("Database error")
```

## Управление уровнями логирования

```go
// Получение текущего уровня
currentLevel := logger.GetLevel()
fmt.Printf("Current level: %s\n", currentLevel)

// Установка нового уровня
if err := logger.SetLevel("warn"); err != nil {
    log.Fatal(err)
}

// Проверка уровня экземпляра логгера
if myLogger.GetLevel() <= zerolog.InfoLevel {
    // логирование info активно
}
```

## Продвинутые возможности

### Прямой доступ к zerolog

Для сложных случаев использования можно получить доступ к базовому zerolog.Logger:

```go
rawLogger := logger.GetGlobal().Raw()
rawLogger.Info().
    Dict("user", zerolog.Dict().
        Str("name", "John").
        Int("age", 30)).
    Msg("Complex nested structure")
```

### Условное логирование для производительности

```go
// Создаем событие без выполнения дорогих операций
event := logger.Debug()
if event != nil { // проверяем, что событие будет записано
    expensiveData := performExpensiveOperation()
    event.Str("data", expensiveData).Msg("Debug info")
}
```

## Типы полей

Пакет поддерживает все основные типы полей zerolog:

```go
logger.Info().
    Str("string_field", "value").                    // string
    Int("int_field", 42).                           // int
    Int64("int64_field", 123456789).                // int64
    Float64("float_field", 3.14).                   // float64
    Bool("bool_field", true).                       // bool
    Time("time_field", time.Now()).                 // time.Time
    Dur("duration_field", 5*time.Second).           // time.Duration
    Interface("interface_field", complexObject).     // any
    Err(err).                                       // error
    Msg("Message with all field types")
```

## Примеры использования в приложениях

### HTTP сервер

```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    reqLogger := logger.WithFields(map[string]any{
        "method":     r.Method,
        "path":       r.URL.Path,
        "request_id": generateRequestID(),
    })
    
    reqLogger.Info().Msg("Request received")
    
    // обработка запроса...
    
    reqLogger.Info().
        Int("status_code", 200).
        Dur("response_time", time.Since(startTime)).
        Msg("Request completed")
}
```

### Обработка ошибок

```go
func processData(data []byte) error {
    funcLogger := logger.WithField("function", "processData")
    
    funcLogger.Info().Int("data_size", len(data)).Msg("Starting data processing")
    
    if len(data) == 0 {
        err := errors.New("empty data")
        funcLogger.Error().Err(err).Msg("Processing failed")
        return err
    }
    
    // обработка данных...
    
    funcLogger.Info().Msg("Data processing completed successfully")
    return nil
}
```

### Инициализация в main

```go
func main() {
    // Конфигурация из переменных окружения или конфиг файла
    cfg := logger.Config{
        Level:      getEnv("LOG_LEVEL", "info"),
        Format:     getEnv("LOG_FORMAT", "json"),
        Output:     getEnv("LOG_OUTPUT", "stdout"),
        CallerInfo: getEnvBool("LOG_CALLER", false),
    }
    
    if err := logger.Init(cfg); err != nil {
        log.Fatalf("Failed to initialize logger: %v", err)
    }
    
    logger.Info().Str("version", version).Msg("Application starting")
    
    // запуск приложения...
}
```

## Миграция с zerolog

Для миграции с прямого использования zerolog:

1. Замените импорты:
   ```go
   // Было
   import "github.com/rs/zerolog"
   
   // Стало
   import "gitlab.com/zynero/shared/logger"
   ```

2. Замените создание логгера:
   ```go
   // Было
   logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
   
   // Стало
   logger.Init(logger.Config{}) // или используйте глобальные функции
   ```

3. Используйте глобальные функции или создавайте экземпляры:
   ```go
   // Было
   logger.Info().Msg("message")
   
   // Стало (глобальные функции)
   logger.Info().Msg("message")
   
   // Или (экземпляр)
   myLogger, _ := logger.New(config)
   myLogger.Info().Msg("message")
   ```

## Лучшие практики

1. **Инициализация**: Инициализируйте логгер один раз в начале приложения
2. **Контекстные логгеры**: Создавайте логгеры с контекстом для компонентов
3. **Уровни логирования**: Используйте соответствующие уровни для разных типов сообщений
4. **Поля**: Добавляйте структурированную информацию через поля, а не в текст сообщения
5. **Производительность**: Используйте условное логирование для дорогих операций
6. **Ошибки**: Всегда используйте .Err() для логирования ошибок

## Лицензия

Этот пакет распространяется под той же лицензией, что и основной проект. 