package main

import (
	"context"
	"errors"
	"time"

	"gitlab.com/zynero/shared/logger"
)

func main() {
	// 1. Инициализация логгера с конфигурацией
	cfg := logger.Config{
		Level:      "debug",
		Format:     "console", // или "json"
		Output:     "stdout",
		TimeFormat: time.RFC3339,
		CallerInfo: true, // добавлять информацию о месте вызова
	}

	// Инициализируем глобальный логгер
	if err := logger.Init(cfg); err != nil {
		panic(err)
	}

	// 2. Простое логирование (глобальные функции)
	logger.Info().Msg("Application started")
	logger.Debug().Str("version", "1.0.0").Msg("Debug information")
	logger.Warn().Msg("This is a warning")

	// 3. Форматированное логирование
	logger.Infof("User %s logged in", "john_doe")
	logger.Debugf("Processing %d items", 42)

	// 4. Логирование с дополнительными полями
	logger.Info().
		Str("user_id", "12345").
		Int("items_count", 10).
		Bool("is_premium", true).
		Msg("User action completed")

	// 5. Работа с ошибками
	err := errors.New("something went wrong")
	logger.Error().Err(err).Msg("Error occurred")
	logger.WithError(err).Info().Msg("Continuing with error context")

	// 6. Создание логгера с контекстом
	userLogger := logger.WithField("user_id", "12345")
	userLogger.Info().Msg("User-specific log entry")

	// Логгер с несколькими полями
	serviceLogger := logger.WithFields(map[string]interface{}{
		"service":    "user-service",
		"version":    "1.2.3",
		"request_id": "req-123",
	})
	serviceLogger.Info().Msg("Service operation completed")

	// 7. Использование контекста
	ctx := context.Background()
	ctxLogger := logger.WithContext(ctx)
	ctxLogger.Info().Msg("Operation with context")

	// 8. Построитель логгера с множеством полей
	logger.With().
		Str("component", "database").
		Int64("connection_id", 12345).
		Dur("query_time", 150*time.Millisecond).
		Float64("cpu_usage", 23.5).
		Time("timestamp", time.Now()).
		Interface("metadata", map[string]string{"region": "us-east-1"}).
		Logger().
		Info().Msg("Database query completed")

	// 9. Создание отдельного экземпляра логгера
	fileCfg := logger.Config{
		Level:  "error",
		Format: "json",
		Output: "error.log", // логирование в файл
	}

	fileLogger, err := logger.New(fileCfg)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create file logger")
		return
	}

	fileLogger.Error().
		Str("error_type", "database_connection").
		Str("database", "postgres").
		Msg("Database connection failed")

	// 10. Работа с уровнями логирования
	logger.Infof("Current log level: %s", logger.GetLevel())

	// Изменение уровня логирования
	if err := logger.SetLevel("warn"); err != nil {
		logger.Error().Err(err).Msg("Failed to set log level")
	}

	logger.Debug().Msg("This debug message won't be shown")
	logger.Warn().Msg("This warning will be shown")

	// 11. Демонстрация всех уровней логирования
	logger.SetLevel("trace")

	logger.Trace().Msg("Trace level message")
	logger.Debug().Msg("Debug level message")
	logger.Info().Msg("Info level message")
	logger.Warn().Msg("Warn level message")
	logger.Error().Msg("Error level message")

	// Fatal и Panic закомментированы, так как они завершают программу
	// logger.Fatal().Msg("Fatal level message") // завершит программу
	// logger.Panic().Msg("Panic level message") // вызовет панику

	// 12. Продвинутое использование - прямой доступ к zerolog
	rawLogger := logger.GetGlobal().Raw()
	rawLogger.Info().Msg("Direct zerolog access when needed")

	// 13. Производительность - использование событий без выполнения
	event := logger.Debug() // создаем событие
	if event != nil {       // проверяем, что оно не nil (уровень активен)
		// выполняем дорогостоящие операции только если событие будет записано
		expensiveOperation := func() string {
			time.Sleep(1 * time.Millisecond) // имитация дорогой операции
			return "expensive result"
		}
		event.Str("result", expensiveOperation()).Msg("Expensive operation completed")
	}

	// 14. Пример обработки пользователей
	users := []string{"user1", "user2", "invalid", "user3"}
	for _, userID := range users {
		if err := processUser(userID); err != nil {
			logger.Error().Err(err).Str("user_id", userID).Msg("Failed to process user")
		}
	}

	logger.Info().Msg("Example completed successfully")
}

// Пример функции с логированием
func processUser(userID string) error {
	// Создаем логгер с контекстом для этой функции
	funcLogger := logger.WithField("user_id", userID).WithField("function", "processUser")

	funcLogger.Info().Msg("Starting user processing")

	// Имитация работы
	time.Sleep(100 * time.Millisecond)

	// Проверка на ошибку
	if userID == "invalid" {
		err := errors.New("invalid user ID")
		funcLogger.Error().Err(err).Msg("User processing failed")
		return err
	}

	funcLogger.Info().
		Dur("processing_time", 100*time.Millisecond).
		Msg("User processing completed successfully")

	return nil
}

func init() {
	// Пример инициализации в init функции
	logger.Info().Msg("Package initialized")
}
