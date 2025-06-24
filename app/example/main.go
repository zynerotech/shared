package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"gitlab.com/zynero/shared/app"
	"gitlab.com/zynero/shared/logger"
)

// AppConfig представляет конфигурацию приложения
type AppConfig struct {
	Logger    logger.Config        `mapstructure:"logger"`
	GlobalLog *logger.GlobalConfig `mapstructure:"global_logger"`
	// Другие конфигурации...
}

func (c *AppConfig) Validate() error {
	return nil
}

func (c *AppConfig) LoggerConfig() logger.Config {
	return c.Logger
}

func (c *AppConfig) GlobalLoggerConfig() *logger.GlobalConfig {
	return c.GlobalLog
}

// Заглушки для других конфигураций
func (c *AppConfig) MetricsConfig() interface{}     { return nil }
func (c *AppConfig) HealthcheckConfig() interface{} { return nil }
func (c *AppConfig) ServerConfig() interface{}      { return nil }
func (c *AppConfig) DatabaseConfig() interface{}    { return nil }
func (c *AppConfig) CacheConfig() interface{}       { return nil }
func (c *AppConfig) KafkaConfig() interface{}       { return nil }
func (c *AppConfig) GRPCConfig() interface{}        { return nil }

func main() {
	// Способ 1: Использование с автоматической глобальной конфигурацией
	cfg := &AppConfig{
		Logger: logger.Config{
			Level:      "debug",
			Format:     "console",
			Output:     "stdout",
			CallerInfo: true,
		},
	}

	// Инициализируем приложение с автоматической глобальной конфигурацией
	application, err := app.BootstrapWithGlobalConfig(cfg, "config.yaml", "user-service", "1.0.0")
	if err != nil {
		log.Fatalf("Failed to bootstrap application: %v", err)
	}
	defer application.Close()

	// Способ 2: Использование с предустановленной глобальной конфигурацией
	/*
		cfg := &AppConfig{
			Logger: logger.Config{
				Level:      "info",
				Format:     "json",
				Output:     "stdout",
			},
			GlobalLog: &logger.GlobalConfig{
				Logger: logger.Config{
					Level:      "debug",
					Format:     "console",
					Output:     "stdout",
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
					"cluster":     "main",
				},
				Components: map[string]logger.ComponentConfig{
					"database": {
						Level: "warn",
						Fields: map[string]any{
							"db_type": "postgres",
						},
					},
					"api": {
						Level: "debug",
						Fields: map[string]any{
							"api_version": "v1",
						},
					},
				},
			},
		}

		application, err := app.BootstrapWithConfig(cfg, "config.yaml")
		if err != nil {
			log.Fatalf("Failed to bootstrap application: %v", err)
		}
		defer application.Close()
	*/

	// Демонстрация использования логгера в разных компонентах
	demoLogging()

	// Ожидаем сигнала завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info().Msg("Application shutdown requested")
}

func demoLogging() {
	// Использование глобального логгера
	logger.Info().Msg("Application started successfully")

	// Использование логгеров компонентов
	dbLogger := logger.Component("database")
	dbLogger.Info().Msg("Database connection established")
	dbLogger.Warn().Msg("Slow query detected")

	apiLogger := logger.Component("api")
	apiLogger.Debug().Msg("Processing API request")
	apiLogger.Info().Str("endpoint", "/users").Int("status", 200).Msg("API request completed")

	cacheLogger := logger.Component("cache")
	cacheLogger.Info().Msg("Cache initialized")

	kafkaLogger := logger.Component("kafka")
	kafkaLogger.Info().Msg("Kafka producer ready")

	grpcLogger := logger.Component("grpc")
	grpcLogger.Info().Msg("gRPC server listening")

	// Демонстрация динамического обновления
	logger.Info().Msg("=== Updating global fields ===")
	logger.UpdateGlobalFields(map[string]any{
		"request_id": "req-12345",
		"user_id":    "user-67890",
	})

	logger.Info().Msg("Message with updated global fields")

	// Демонстрация изменения уровня компонента
	logger.Info().Msg("=== Changing component level ===")
	logger.SetComponentLevel("database", "debug")

	newDBLogger := logger.Component("database")
	newDBLogger.Debug().Msg("This debug message will now show!")

	// Просмотр информации о компонентах
	components := logger.ListComponents()
	logger.Info().Interface("registered_components", components).Msg("All registered components")

	for _, comp := range components {
		level := logger.GetComponentLevel(comp)
		logger.Info().Str("component", comp).Str("level", level).Msg("Component configuration")
	}
}
