package main

import (
	"log"
	"os"
	"time"

	"gitlab.com/zynero/shared/logger"
)

func main() {
	// Демонстрация интеграции глобального логгера в bootstrap

	// Способ 1: Простая инициализация с автоматической глобальной конфигурацией
	logger.Info().Msg("=== Демонстрация интеграции с bootstrap ===")

	// Создаем базовую конфигурацию
	cfg := logger.Config{
		Level:      "debug",
		Format:     "console",
		Output:     "stdout",
		CallerInfo: true,
	}

	// Инициализируем глобальную конфигурацию (как это делает bootstrap)
	globalCfg := logger.GlobalConfig{
		Logger: cfg,
		Application: logger.ApplicationInfo{
			Name:        "demo-service",
			Version:     "1.0.0",
			Environment: getEnvironment(),
			Instance:    getHostname(),
		},
		GlobalFields: map[string]any{
			"service_type": "microservice",
			"startup_time": time.Now().Format(time.RFC3339),
		},
		Components: map[string]logger.ComponentConfig{
			"app": {
				Level: "info",
			},
			"database": {
				Level: "warn",
				Fields: map[string]any{
					"component_type": "database",
				},
			},
			"api": {
				Level: "debug",
				Fields: map[string]any{
					"component_type": "http_api",
				},
			},
		},
	}

	// Инициализируем глобальный логгер (как это делает bootstrap)
	if err := logger.InitGlobal(globalCfg); err != nil {
		log.Fatalf("Failed to initialize global logger: %v", err)
	}

	// Теперь демонстрируем, как это будет работать в bootstrap
	demoBootstrapIntegration()

	logger.Info().Msg("=== Демонстрация завершена ===")
}

func demoBootstrapIntegration() {
	// Имитация инициализации компонентов в bootstrap

	// 1. Логирование инициализации приложения
	appLogger := logger.Component("app")
	appLogger.Info().Msg("Initializing application components")

	// 2. Имитация инициализации базы данных
	dbLogger := logger.Component("database")
	dbLogger.Info().Msg("Connecting to database")
	dbLogger.Warn().Msg("Database connection established")
	// Debug сообщение не покажется, так как уровень = warn
	dbLogger.Debug().Msg("This debug message won't show")

	// 3. Имитация инициализации API
	apiLogger := logger.Component("api")
	apiLogger.Debug().Msg("Setting up API routes")
	apiLogger.Info().Msg("API server initialized")

	// 4. Глобальное логирование
	logger.Info().Msg("All components initialized successfully")

	// 5. Демонстрация динамического управления
	logger.Info().Msg("=== Dynamic management demo ===")

	// Включаем debug для базы данных
	logger.SetComponentLevel("database", "debug")

	// Теперь debug сообщения будут показываться
	newDBLogger := logger.Component("database")
	newDBLogger.Debug().Msg("Now this debug message will show!")

	// Обновляем глобальные поля
	logger.UpdateGlobalFields(map[string]any{
		"request_id": "bootstrap-demo-123",
		"phase":      "initialization",
	})

	logger.Info().Msg("Message with updated global fields")

	// Просмотр конфигурации
	components := logger.ListComponents()
	logger.Info().Interface("components", components).Msg("Registered components")

	currentConfig := logger.GetGlobalConfig()
	if currentConfig != nil {
		logger.Info().
			Str("app_name", currentConfig.Application.Name).
			Str("app_version", currentConfig.Application.Version).
			Str("environment", currentConfig.Application.Environment).
			Int("global_fields_count", len(currentConfig.GlobalFields)).
			Int("components_count", len(currentConfig.Components)).
			Msg("Current global configuration")
	}
}

func getEnvironment() string {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = os.Getenv("ENV")
	}
	if env == "" {
		env = "development"
	}
	return env
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}
