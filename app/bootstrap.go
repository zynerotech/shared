package app

import (
	"fmt"
	"os"
	"time"

	platformcache "gitlab.com/zynero/shared/cache"
	platformconfig "gitlab.com/zynero/shared/config"
	platformdatabase "gitlab.com/zynero/shared/database"
	platformgrpc "gitlab.com/zynero/shared/grpc"
	platformhealthcheck "gitlab.com/zynero/shared/healthcheck"
	platformlogger "gitlab.com/zynero/shared/logger"
	platformmetrics "gitlab.com/zynero/shared/metrics"
	platformserver "gitlab.com/zynero/shared/server"
	"gitlab.com/zynero/shared/transport/kafka"
)

// ConfigProvider describes configuration required to bootstrap common
// infrastructure components. It should be implemented by a service specific
// configuration struct.
type ConfigProvider interface {
	Validate() error
	LoggerConfig() platformlogger.Config
	// GlobalLoggerConfig возвращает расширенную глобальную конфигурацию логгера
	GlobalLoggerConfig() *platformlogger.GlobalConfig
	MetricsConfig() platformmetrics.Config
	HealthcheckConfig() platformhealthcheck.Config
	ServerConfig() platformserver.Config
	DatabaseConfig() platformdatabase.Config
	CacheConfig() platformcache.Config
	KafkaConfig() kafka.Config
	GRPCConfig() platformgrpc.Config
}

// App contains initialized shared components used across applications.
type App struct {
	Config         ConfigProvider
	Logger         *platformlogger.Logger
	Metrics        *platformmetrics.Metrics
	Healthcheck    *platformhealthcheck.Healthcheck
	Server         *platformserver.Server
	GRPCServer     *platformgrpc.Server
	Database       *platformdatabase.Database
	Cache          platformcache.Cache
	EventPublisher *kafka.KafkaEventPublisher
}

// New initializes all common infrastructure services based on provided
// configuration.
func New(cfg ConfigProvider) (*App, error) {
	// Инициализируем логгер с поддержкой глобальной конфигурации
	var logger *platformlogger.Logger
	var err error

	// Проверяем, есть ли глобальная конфигурация
	if globalCfg := cfg.GlobalLoggerConfig(); globalCfg != nil {
		// Используем расширенную глобальную конфигурацию
		if err := platformlogger.InitGlobal(*globalCfg); err != nil {
			return nil, fmt.Errorf("init global logger: %w", err)
		}
		logger = platformlogger.GetGlobal()
	} else {
		// Используем старый способ для обратной совместимости
		logger, err = platformlogger.New(cfg.LoggerConfig())
		if err != nil {
			return nil, fmt.Errorf("init logger: %w", err)
		}
		platformlogger.SetGlobal(logger)
	}

	// Логируем информацию о запуске приложения
	appLogger := platformlogger.Component("app")
	appLogger.Info().Msg("Initializing application components")

	metrics, err := platformmetrics.New(cfg.MetricsConfig())
	if err != nil {
		return nil, fmt.Errorf("init metrics: %w", err)
	}
	appLogger.Info().Msg("Metrics initialized")

	health, err := platformhealthcheck.New(cfg.HealthcheckConfig())
	if err != nil {
		return nil, fmt.Errorf("init healthcheck: %w", err)
	}
	appLogger.Info().Msg("Healthcheck initialized")

	server, err := platformserver.New(cfg.ServerConfig())
	if err != nil {
		return nil, fmt.Errorf("init server: %w", err)
	}
	appLogger.Info().Msg("HTTP server initialized")

	cache, err := platformcache.New(cfg.CacheConfig())
	if err != nil {
		return nil, fmt.Errorf("init cache: %w", err)
	}
	appLogger.Info().Msg("Cache initialized")

	db, err := platformdatabase.New(cfg.DatabaseConfig())
	if err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}
	appLogger.Info().Msg("Database initialized")

	producer, err := kafka.NewProducer(cfg.KafkaConfig())
	if err != nil {
		return nil, fmt.Errorf("init kafka producer: %w", err)
	}
	publisher := kafka.NewKafkaEventPublisher(producer, cfg.KafkaConfig().Producer.Topic)
	appLogger.Info().Msg("Kafka producer initialized")

	grpcServer, err := platformgrpc.NewServer(cfg.GRPCConfig(), logger, nil)
	if err != nil {
		return nil, fmt.Errorf("init grpc server: %w", err)
	}
	appLogger.Info().Msg("gRPC server initialized")

	appLogger.Info().Msg("All application components initialized successfully")

	return &App{
		Config:         cfg,
		Logger:         logger,
		Metrics:        metrics,
		Healthcheck:    health,
		Server:         server,
		GRPCServer:     grpcServer,
		Database:       db,
		Cache:          cache,
		EventPublisher: publisher,
	}, nil
}

// Close stops metrics, health checks and closes database connections.
func (a *App) Close() error {
	if a == nil {
		return nil
	}

	appLogger := platformlogger.Component("app")
	appLogger.Info().Msg("Shutting down application components")

	a.Database.Close()
	appLogger.Info().Msg("Database connection closed")

	if err := a.Metrics.Stop(); err != nil {
		appLogger.Error().Err(err).Msg("Failed to stop metrics")
		return err
	}
	appLogger.Info().Msg("Metrics stopped")

	if err := a.Healthcheck.Stop(); err != nil {
		appLogger.Error().Err(err).Msg("Failed to stop healthcheck")
		return err
	}
	appLogger.Info().Msg("Healthcheck stopped")

	appLogger.Info().Msg("Application shutdown completed")
	return nil
}

// BootstrapWithConfig загружает конфигурацию из файла и инициализирует приложение.
// Эта функция объединяет загрузку конфигурации и инициализацию в одном месте,
// что устраняет дублирование кода в точках входа.
func BootstrapWithConfig(cfg ConfigProvider, configPath string) (*App, error) {
	// Загружаем конфигурацию из файла
	if err := platformconfig.Load(cfg, configPath); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Инициализируем приложение
	return New(cfg)
}

// BootstrapWithGlobalConfig инициализирует приложение с расширенной глобальной конфигурацией логгера.
// Эта функция позволяет использовать все новые возможности логгера.
func BootstrapWithGlobalConfig(cfg ConfigProvider, configPath string, appName, appVersion string) (*App, error) {
	// Загружаем конфигурацию из файла
	if err := platformconfig.Load(cfg, configPath); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Если глобальная конфигурация не установлена, создаем её
	if cfg.GlobalLoggerConfig() == nil {
		// Получаем hostname для instance
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}

		// Создаем расширенную конфигурацию на основе базовой
		globalCfg := platformlogger.GlobalConfig{
			Logger: cfg.LoggerConfig(),
			Application: platformlogger.ApplicationInfo{
				Name:        appName,
				Version:     appVersion,
				Environment: getEnvironment(),
				Instance:    hostname,
			},
			GlobalFields: map[string]any{
				"service_type": "microservice",
				"startup_time": time.Now().Format(time.RFC3339),
			},
			Components: map[string]platformlogger.ComponentConfig{
				"app": {
					Level: "info",
				},
				"database": {
					Level: "warn", // для БД только важные сообщения
					Fields: map[string]any{
						"component_type": "database",
					},
				},
				"cache": {
					Level: "info",
					Fields: map[string]any{
						"component_type": "cache",
					},
				},
				"kafka": {
					Level: "info",
					Fields: map[string]any{
						"component_type": "message_broker",
					},
				},
				"grpc": {
					Level: "info",
					Fields: map[string]any{
						"component_type": "grpc_server",
					},
				},
				"http": {
					Level: "info",
					Fields: map[string]any{
						"component_type": "http_server",
					},
				},
			},
		}

		// Создаем новый ConfigProvider с глобальной конфигурацией
		enhancedCfg := &enhancedConfigProvider{
			ConfigProvider: cfg,
			globalConfig:   &globalCfg,
		}

		return New(enhancedCfg)
	}

	// Инициализируем приложение с существующей глобальной конфигурацией
	return New(cfg)
}

// enhancedConfigProvider оборачивает ConfigProvider для добавления глобальной конфигурации
type enhancedConfigProvider struct {
	ConfigProvider
	globalConfig *platformlogger.GlobalConfig
}

func (e *enhancedConfigProvider) GlobalLoggerConfig() *platformlogger.GlobalConfig {
	return e.globalConfig
}

// getEnvironment определяет окружение приложения
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
