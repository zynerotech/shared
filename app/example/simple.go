package main

import (
	"log"

	"gitlab.com/zynero/shared/app"
	platformlogger "gitlab.com/zynero/shared/logger"
)

// SimpleConfig представляет минимальную конфигурацию только с логгером
type SimpleConfig struct {
	Logger platformlogger.Config `mapstructure:"logger"`
}

// Validate проверяет корректность конфигурации
func (c SimpleConfig) Validate() error {
	return nil
}

// LoggerConfig возвращает конфигурацию логгера (обязательный)
func (c SimpleConfig) LoggerConfig() platformlogger.Config {
	return c.Logger
}

func runSimpleExample() {
	// Создаем минимальную конфигурацию
	cfg := SimpleConfig{
		Logger: platformlogger.Config{
			Level:      "info",
			Format:     "console",
			Output:     "stdout",
			CallerInfo: true,
		},
	}

	// Инициализируем приложение только с логгером
	application, err := app.NewWithLogger(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer application.Close()

	// Используем логгер
	platformlogger.Info().Msg("Simple application started")
	platformlogger.Info().Str("component", "main").Msg("Application is ready")

	// Проверяем, что другие компоненты не инициализированы
	if application.Metrics == nil {
		platformlogger.Info().Msg("Metrics component is not initialized (as expected)")
	}

	if application.Database == nil {
		platformlogger.Info().Msg("Database component is not initialized (as expected)")
	}

	if application.Server == nil {
		platformlogger.Info().Msg("HTTP server component is not initialized (as expected)")
	}

	platformlogger.Info().Msg("Simple example completed successfully")
}
