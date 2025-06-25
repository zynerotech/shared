package app

import (
	"context"
	"errors"
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

// initOptionalComponent initializes optional component based on configuration
// provided by OptionalConfigProvider. It appends initialization errors to the
// builder and logs successful initialization.
func initOptionalComponent[T any, C any](b *AppBuilder, field *T, getCfg func(OptionalConfigProvider) *C, initFn func(C) (T, error), name, successMsg string) {
	optCfg, ok := b.config.(OptionalConfigProvider)
	if !ok {
		return
	}

	cfg := getCfg(optCfg)
	if cfg == nil {
		return
	}

	component, err := initFn(*cfg)
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("init %s: %w", name, err))
		return
	}

	*field = component
	platformlogger.Info().Msg(successMsg)
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
	initOptionalComponent(b, &b.metrics, func(o OptionalConfigProvider) *platformmetrics.Config { return o.MetricsConfig() }, func(cfg platformmetrics.Config) (*platformmetrics.Metrics, error) {
		return platformmetrics.New(cfg)
	}, "metrics", "Metrics initialized")
	return b
}

// WithHealthcheck initializes healthcheck if configuration is provided
func (b *AppBuilder) WithHealthcheck() *AppBuilder {
	if b.healthcheck != nil {
		return b
	}
	initOptionalComponent(b, &b.healthcheck, func(o OptionalConfigProvider) *platformhealthcheck.Config { return o.HealthcheckConfig() }, func(cfg platformhealthcheck.Config) (*platformhealthcheck.Healthcheck, error) {
		return platformhealthcheck.New(cfg)
	}, "healthcheck", "Healthcheck initialized")
	return b
}

// WithServer initializes HTTP server if configuration is provided
func (b *AppBuilder) WithServer() *AppBuilder {
	if b.server != nil {
		return b
	}
	initOptionalComponent(b, &b.server, func(o OptionalConfigProvider) *platformserver.Config { return o.ServerConfig() }, func(cfg platformserver.Config) (*platformserver.Server, error) {
		return platformserver.New(cfg)
	}, "server", "HTTP server initialized")
	return b
}

// WithDatabase initializes database if configuration is provided
func (b *AppBuilder) WithDatabase() *AppBuilder {
	if b.database != nil {
		return b
	}
	initOptionalComponent(b, &b.database, func(o OptionalConfigProvider) *platformdatabase.Config { return o.DatabaseConfig() }, func(cfg platformdatabase.Config) (*platformdatabase.Database, error) {
		return platformdatabase.New(cfg)
	}, "database", "Database initialized")
	return b
}

// WithCache initializes cache if configuration is provided
func (b *AppBuilder) WithCache() *AppBuilder {
	if b.cache != nil {
		return b
	}
	initOptionalComponent(b, &b.cache, func(o OptionalConfigProvider) *platformcache.Config { return o.CacheConfig() }, func(cfg platformcache.Config) (platformcache.Cache, error) {
		return platformcache.New(cfg)
	}, "cache", "Cache initialized")
	return b
}

// WithKafka initializes Kafka producer and event publisher if configuration is provided
func (b *AppBuilder) WithKafka() *AppBuilder {
	if b.eventPublisher != nil {
		return b
	}
	initOptionalComponent(b, &b.eventPublisher, func(o OptionalConfigProvider) *kafka.Config { return o.KafkaConfig() }, func(cfg kafka.Config) (*kafka.KafkaEventPublisher, error) {
		producer, err := kafka.NewProducer(cfg)
		if err != nil {
			return nil, err
		}
		return kafka.NewKafkaEventPublisher(producer, cfg.Producer.Topic), nil
	}, "kafka producer", "Kafka producer initialized")
	return b
}

// WithGRPC initializes gRPC server if configuration is provided
func (b *AppBuilder) WithGRPC() *AppBuilder {
	if b.grpcServer != nil {
		return b
	}
	initOptionalComponent(b, &b.grpcServer, func(o OptionalConfigProvider) *platformgrpc.Config { return o.GRPCConfig() }, func(cfg platformgrpc.Config) (*platformgrpc.Server, error) {
		return platformgrpc.NewServer(cfg, b.logger, nil)
	}, "grpc server", "gRPC server initialized")
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
		return nil, fmt.Errorf("failed to build app: %w", errors.Join(b.errors...))
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

	if a.Server != nil {
		if err := a.Server.Stop(); err != nil {
			platformlogger.Error().Err(err).Msg("Failed to stop HTTP server")
			return err
		}
		platformlogger.Info().Msg("HTTP server stopped")
	}

	if a.GRPCServer != nil {
		if err := a.GRPCServer.Stop(context.Background()); err != nil {
			platformlogger.Error().Err(err).Msg("Failed to stop gRPC server")
			return err
		}
		platformlogger.Info().Msg("gRPC server stopped")
	}

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

	if a.Cache != nil {
		if err := a.Cache.Close(); err != nil {
			platformlogger.Error().Err(err).Msg("Failed to close cache")
			return err
		}
		platformlogger.Info().Msg("Cache closed")
	}

	if a.EventPublisher != nil {
		if err := a.EventPublisher.Close(); err != nil {
			platformlogger.Error().Err(err).Msg("Failed to close event publisher")
			return err
		}
		platformlogger.Info().Msg("Event publisher closed")
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
