package main

import (
	"errors"
	"os"
	"time"

	"gitlab.com/zynero/shared/logger"
)

func main() {
	// 🌟 Новый способ: Глобальная конфигурация приложения
	globalCfg := logger.GlobalConfig{
		// Основные настройки логгера
		Logger: logger.Config{
			Level:      "debug",
			Format:     "console", // для наглядности в примере
			Output:     "stdout",
			TimeFormat: time.RFC3339,
			CallerInfo: true,
		},

		// Информация о приложении (будет добавлена ко всем сообщениям)
		Application: logger.ApplicationInfo{
			Name:        "user-service",
			Version:     "1.2.3",
			Environment: "production",
			Instance:    getHostname(),
		},

		// Глобальные поля, которые будут во всех сообщениях
		GlobalFields: map[string]any{
			"service_type": "microservice",
			"region":       "us-east-1",
			"cluster":      "main",
		},

		// Настройки для конкретных компонентов
		Components: map[string]logger.ComponentConfig{
			"database": {
				Level: "warn", // для БД только важные сообщения
				Fields: map[string]any{
					"db_type":    "postgres",
					"connection": "primary",
				},
			},
			"auth": {
				Level: "info",
				Fields: map[string]any{
					"auth_provider": "oauth2",
				},
			},
			"api": {
				Level: "debug", // для API детальное логирование
				Fields: map[string]any{
					"api_version": "v1",
				},
			},
		},
	}

	// Инициализируем глобальную конфигурацию
	if err := logger.InitGlobal(globalCfg); err != nil {
		panic(err)
	}

	logger.Info().Msg("=== Демонстрация глобальной конфигурации ===")

	// 1. Простое логирование - автоматически включает глобальные поля
	logger.Info().Msg("Application started")
	logger.Error().Msg("This includes all global fields automatically")

	// 2. Логирование по компонентам с их настройками
	databaseLogger := logger.Component("database")
	databaseLogger.Info().Msg("This will be logged as WARN level due to component config")
	databaseLogger.Warn().Msg("Database connection established") // Это будет показано
	databaseLogger.Debug().Msg("This debug won't show - component level is WARN")

	authLogger := logger.Component("auth")
	authLogger.Info().Msg("User authentication attempt")
	authLogger.Error().Str("user_id", "12345").Msg("Authentication failed")

	apiLogger := logger.Component("api")
	apiLogger.Debug().Msg("This debug message will show - component level is DEBUG")
	apiLogger.Info().Str("endpoint", "/users").Int("status", 200).Msg("API request handled")

	// 3. Использование в разных функциях/пакетах
	simulateUserService()
	simulatePaymentService()
	simulateNotificationService()

	// 4. Динамическое обновление глобальных полей
	logger.Info().Msg("=== Updating global fields dynamically ===")
	logger.UpdateGlobalFields(map[string]any{
		"feature_flag": "new_feature_enabled",
		"experiment":   "A/B-test-123",
	})

	logger.Info().Msg("Message with updated global fields")

	// 5. Управление уровнями компонентов во время выполнения
	logger.Info().Msg("=== Dynamic component level management ===")
	logger.SetComponentLevel("database", "debug") // Включаем debug для БД

	// Новый логгер БД теперь будет с debug уровнем
	newDBLogger := logger.Component("database")
	newDBLogger.Debug().Msg("Now this debug message will show!")

	// 6. Просмотр информации о компонентах
	components := logger.ListComponents()
	logger.Info().Interface("registered_components", components).Msg("All registered components")

	for _, comp := range components {
		level := logger.GetComponentLevel(comp)
		logger.Info().Str("component", comp).Str("level", level).Msg("Component level")
	}

	// 7. Получение текущей глобальной конфигурации
	currentConfig := logger.GetGlobalConfig()
	if currentConfig != nil {
		logger.Info().
			Str("app_name", currentConfig.Application.Name).
			Str("app_version", currentConfig.Application.Version).
			Int("global_fields_count", len(currentConfig.GlobalFields)).
			Int("components_count", len(currentConfig.Components)).
			Msg("Current global configuration")
	}

	logger.Info().Msg("=== Example completed successfully ===")
}

// Имитация использования в разных сервисах
func simulateUserService() {
	// В реальном приложении это был бы отдельный пакет
	userLogger := logger.Component("user-service")
	userLogger.Info().Str("operation", "create_user").Msg("Creating new user")
	userLogger.Warn().Str("user_id", "user123").Msg("User validation warning")
}

func simulatePaymentService() {
	paymentLogger := logger.Component("payment")
	paymentLogger.Info().
		Str("payment_id", "pay_123").
		Float64("amount", 99.99).
		Str("currency", "USD").
		Msg("Processing payment")

	// Симуляция ошибки
	err := errors.New("insufficient funds")
	paymentLogger.Error().
		Err(err).
		Str("payment_id", "pay_123").
		Msg("Payment processing failed")
}

func simulateNotificationService() {
	notificationLogger := logger.Component("notification")
	notificationLogger.Info().
		Str("type", "email").
		Str("recipient", "user@example.com").
		Msg("Sending notification")

	notificationLogger.Debug().
		Str("template", "welcome_email").
		Msg("Using email template")
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}
