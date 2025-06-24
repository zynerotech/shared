package app

import (
	"fmt"
	platformcache "gitlab.com/zynero/shared/cache"
	platformconfig "gitlab.com/zynero/shared/config"
	platformdatabase "gitlab.com/zynero/shared/database"
	platformgrpc "gitlab.com/zynero/shared/grpc"
	platformhealthcheck "gitlab.com/zynero/shared/healthcheck"
	platformlogger "gitlab.com/zynero/shared/logger"
	platformmetrics "gitlab.com/zynero/shared/metrics"
	platformserver "gitlab.com/zynero/shared/server"
	"gitlab.com/zynero/shared/transport/kafka"
	"os"
)

// ConfigProvider describes configuration required to bootstrap common
// infrastructure components. It should be implemented by a service specific
// configuration struct.
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

// New initializes all common infrastructure services based on the provided configuration
func New(cfg ConfigProvider) (*App, error) {
	// Инициализируем логгер с поддержкой глобальной конфигурации
	var logger *platformlogger.Logger
	var err error

	logger, err = platformlogger.New(cfg.LoggerConfig())
	if err != nil {
		return nil, fmt.Errorf("init logger: %w", err)
	}
	platformlogger.SetGlobal(logger)

	// Логируем информацию о запуске приложения
	platformlogger.Info().Msg("Initializing application components")

	metrics, err := platformmetrics.New(cfg.MetricsConfig())
	if err != nil {
		return nil, fmt.Errorf("init metrics: %w", err)
	}
	platformlogger.Info().Msg("Metrics initialized")

	health, err := platformhealthcheck.New(cfg.HealthcheckConfig())
	if err != nil {
		return nil, fmt.Errorf("init healthcheck: %w", err)
	}
	platformlogger.Info().Msg("Healthcheck initialized")

	server, err := platformserver.New(cfg.ServerConfig())
	if err != nil {
		return nil, fmt.Errorf("init server: %w", err)
	}
	platformlogger.Info().Msg("HTTP server initialized")

	cache, err := platformcache.New(cfg.CacheConfig())
	if err != nil {
		return nil, fmt.Errorf("init cache: %w", err)
	}
	platformlogger.Info().Msg("Cache initialized")

	db, err := platformdatabase.New(cfg.DatabaseConfig())
	if err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}
	platformlogger.Info().Msg("Database initialized")

	producer, err := kafka.NewProducer(cfg.KafkaConfig())
	if err != nil {
		return nil, fmt.Errorf("init kafka producer: %w", err)
	}
	publisher := kafka.NewKafkaEventPublisher(producer, cfg.KafkaConfig().Producer.Topic)
	platformlogger.Info().Msg("Kafka producer initialized")

	grpcServer, err := platformgrpc.NewServer(cfg.GRPCConfig(), logger, nil)
	if err != nil {
		return nil, fmt.Errorf("init grpc server: %w", err)
	}
	platformlogger.Info().Msg("gRPC server initialized")

	platformlogger.Info().Msg("All application components initialized successfully")

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

	platformlogger.Info().Msg("Shutting down application components")

	a.Database.Close()
	platformlogger.Info().Msg("Database connection closed")

	if err := a.Metrics.Stop(); err != nil {
		platformlogger.Error().Err(err).Msg("Failed to stop metrics")
		return err
	}
	platformlogger.Info().Msg("Metrics stopped")

	if err := a.Healthcheck.Stop(); err != nil {
		platformlogger.Error().Err(err).Msg("Failed to stop healthcheck")
		return err
	}
	platformlogger.Info().Msg("Healthcheck stopped")

	platformlogger.Info().Msg("Application shutdown completed")
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
