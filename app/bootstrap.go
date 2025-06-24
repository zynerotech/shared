package app

import (
	"fmt"
	"os"

	platformcache "gitlab.com/zynero/shared/cache"
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
}

// OptionalConfigProvider describes optional configuration methods that may not be implemented
// by all services. These methods should return nil if the component is not needed.
type OptionalConfigProvider interface {
	MetricsConfig() *platformmetrics.Config
	HealthcheckConfig() *platformhealthcheck.Config
	ServerConfig() *platformserver.Config
	DatabaseConfig() *platformdatabase.Config
	CacheConfig() *platformcache.Config
	KafkaConfig() *kafka.Config
	GRPCConfig() *platformgrpc.Config
}

// App contains initialized shared components used across applications.
// Only Logger is guaranteed to be present, other components may be nil.
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

// AppBuilder provides a fluent interface for building App instances
type AppBuilder struct {
	config         ConfigProvider
	logger         *platformlogger.Logger
	metrics        *platformmetrics.Metrics
	healthcheck    *platformhealthcheck.Healthcheck
	server         *platformserver.Server
	grpcServer     *platformgrpc.Server
	database       *platformdatabase.Database
	cache          platformcache.Cache
	eventPublisher *kafka.KafkaEventPublisher
	errors         []error
}

// NewBuilder creates a new AppBuilder with the given configuration
func NewBuilder(cfg ConfigProvider) *AppBuilder {
	return &AppBuilder{
		config: cfg,
		errors: make([]error, 0),
	}
}

// WithLogger initializes the logger (required component)
func (b *AppBuilder) WithLogger() *AppBuilder {
	if b.logger != nil {
		return b
	}

	logger, err := platformlogger.New(b.config.LoggerConfig())
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("init logger: %w", err))
		return b
	}

	platformlogger.SetGlobal(logger)
	b.logger = logger
	platformlogger.Info().Msg("Logger initialized")
	return b
}

// WithMetrics initializes metrics if configuration is provided
func (b *AppBuilder) WithMetrics() *AppBuilder {
	if b.metrics != nil {
		return b
	}

	if optCfg, ok := b.config.(OptionalConfigProvider); ok {
		if cfg := optCfg.MetricsConfig(); cfg != nil {
			metrics, err := platformmetrics.New(*cfg)
			if err != nil {
				b.errors = append(b.errors, fmt.Errorf("init metrics: %w", err))
				return b
			}
			b.metrics = metrics
			platformlogger.Info().Msg("Metrics initialized")
		}
	}
	return b
}

// WithHealthcheck initializes healthcheck if configuration is provided
func (b *AppBuilder) WithHealthcheck() *AppBuilder {
	if b.healthcheck != nil {
		return b
	}

	if optCfg, ok := b.config.(OptionalConfigProvider); ok {
		if cfg := optCfg.HealthcheckConfig(); cfg != nil {
			health, err := platformhealthcheck.New(*cfg)
			if err != nil {
				b.errors = append(b.errors, fmt.Errorf("init healthcheck: %w", err))
				return b
			}
			b.healthcheck = health
			platformlogger.Info().Msg("Healthcheck initialized")
		}
	}
	return b
}

// WithServer initializes HTTP server if configuration is provided
func (b *AppBuilder) WithServer() *AppBuilder {
	if b.server != nil {
		return b
	}

	if optCfg, ok := b.config.(OptionalConfigProvider); ok {
		if cfg := optCfg.ServerConfig(); cfg != nil {
			server, err := platformserver.New(*cfg)
			if err != nil {
				b.errors = append(b.errors, fmt.Errorf("init server: %w", err))
				return b
			}
			b.server = server
			platformlogger.Info().Msg("HTTP server initialized")
		}
	}
	return b
}

// WithDatabase initializes database if configuration is provided
func (b *AppBuilder) WithDatabase() *AppBuilder {
	if b.database != nil {
		return b
	}

	if optCfg, ok := b.config.(OptionalConfigProvider); ok {
		if cfg := optCfg.DatabaseConfig(); cfg != nil {
			db, err := platformdatabase.New(*cfg)
			if err != nil {
				b.errors = append(b.errors, fmt.Errorf("init database: %w", err))
				return b
			}
			b.database = db
			platformlogger.Info().Msg("Database initialized")
		}
	}
	return b
}

// WithCache initializes cache if configuration is provided
func (b *AppBuilder) WithCache() *AppBuilder {
	if b.cache != nil {
		return b
	}

	if optCfg, ok := b.config.(OptionalConfigProvider); ok {
		if cfg := optCfg.CacheConfig(); cfg != nil {
			cache, err := platformcache.New(*cfg)
			if err != nil {
				b.errors = append(b.errors, fmt.Errorf("init cache: %w", err))
				return b
			}
			b.cache = cache
			platformlogger.Info().Msg("Cache initialized")
		}
	}
	return b
}

// WithKafka initializes Kafka producer and event publisher if configuration is provided
func (b *AppBuilder) WithKafka() *AppBuilder {
	if b.eventPublisher != nil {
		return b
	}

	if optCfg, ok := b.config.(OptionalConfigProvider); ok {
		if cfg := optCfg.KafkaConfig(); cfg != nil {
			producer, err := kafka.NewProducer(*cfg)
			if err != nil {
				b.errors = append(b.errors, fmt.Errorf("init kafka producer: %w", err))
				return b
			}
			publisher := kafka.NewKafkaEventPublisher(producer, cfg.Producer.Topic)
			b.eventPublisher = publisher
			platformlogger.Info().Msg("Kafka producer initialized")
		}
	}
	return b
}

// WithGRPC initializes gRPC server if configuration is provided
func (b *AppBuilder) WithGRPC() *AppBuilder {
	if b.grpcServer != nil {
		return b
	}

	if optCfg, ok := b.config.(OptionalConfigProvider); ok {
		if cfg := optCfg.GRPCConfig(); cfg != nil {
			grpcServer, err := platformgrpc.NewServer(*cfg, b.logger, nil)
			if err != nil {
				b.errors = append(b.errors, fmt.Errorf("init grpc server: %w", err))
				return b
			}
			b.grpcServer = grpcServer
			platformlogger.Info().Msg("gRPC server initialized")
		}
	}
	return b
}

// WithAll initializes all available components based on configuration
func (b *AppBuilder) WithAll() *AppBuilder {
	return b.WithLogger().
		WithMetrics().
		WithHealthcheck().
		WithServer().
		WithDatabase().
		WithCache().
		WithKafka().
		WithGRPC()
}

// Build creates the App instance and returns any errors that occurred during initialization
func (b *AppBuilder) Build() (*App, error) {
	// Logger is required
	if b.logger == nil {
		b.WithLogger()
	}

	if len(b.errors) > 0 {
		return nil, fmt.Errorf("failed to build app: %v", b.errors)
	}

	platformlogger.Info().Msg("All requested application components initialized successfully")

	return &App{
		Config:         b.config,
		Logger:         b.logger,
		Metrics:        b.metrics,
		Healthcheck:    b.healthcheck,
		Server:         b.server,
		GRPCServer:     b.grpcServer,
		Database:       b.database,
		Cache:          b.cache,
		EventPublisher: b.eventPublisher,
	}, nil
}

// New initializes all common infrastructure services based on the provided configuration
// This is a convenience method that initializes all components (legacy behavior)
func New(cfg ConfigProvider) (*App, error) {
	return NewBuilder(cfg).WithAll().Build()
}

// NewWithLogger initializes only the logger (minimal setup)
func NewWithLogger(cfg ConfigProvider) (*App, error) {
	return NewBuilder(cfg).WithLogger().Build()
}

// Close stops metrics, health checks and closes database connections.
func (a *App) Close() error {
	if a == nil {
		return nil
	}

	platformlogger.Info().Msg("Shutting down application components")

	if a.Database != nil {
		a.Database.Close()
		platformlogger.Info().Msg("Database connection closed")
	}

	if a.Metrics != nil {
		if err := a.Metrics.Stop(); err != nil {
			platformlogger.Error().Err(err).Msg("Failed to stop metrics")
			return err
		}
		platformlogger.Info().Msg("Metrics stopped")
	}

	if a.Healthcheck != nil {
		if err := a.Healthcheck.Stop(); err != nil {
			platformlogger.Error().Err(err).Msg("Failed to stop healthcheck")
			return err
		}
		platformlogger.Info().Msg("Healthcheck stopped")
	}

	platformlogger.Info().Msg("Application shutdown completed")
	return nil
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
