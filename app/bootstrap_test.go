package app

import (
	"context"
	"reflect"
	"testing"
	"time"

	monkey "bou.ke/monkey"
	platformgrpc "gitlab.com/zynero/shared/grpc"
	platformlogger "gitlab.com/zynero/shared/logger"
	platformserver "gitlab.com/zynero/shared/server"
	"gitlab.com/zynero/shared/transport/kafka"
)

// TestConfig представляет тестовую конфигурацию
type TestConfig struct {
	Logger platformlogger.Config `mapstructure:"logger"`
}

// Validate проверяет корректность конфигурации
func (c TestConfig) Validate() error {
	return nil
}

// LoggerConfig возвращает конфигурацию логгера (обязательный)
func (c TestConfig) LoggerConfig() platformlogger.Config {
	return c.Logger
}

// TestOptionalConfig представляет тестовую конфигурацию с опциональными компонентами
type TestOptionalConfig struct {
	TestConfig
}

// MetricsConfig возвращает nil (компонент не нужен)
func (c TestOptionalConfig) MetricsConfig() *platformlogger.Config {
	return nil
}

// HealthcheckConfig возвращает nil (компонент не нужен)
func (c TestOptionalConfig) HealthcheckConfig() *platformlogger.Config {
	return nil
}

// ServerConfig возвращает nil (компонент не нужен)
func (c TestOptionalConfig) ServerConfig() *platformlogger.Config {
	return nil
}

// DatabaseConfig возвращает nil (компонент не нужен)
func (c TestOptionalConfig) DatabaseConfig() *platformlogger.Config {
	return nil
}

// CacheConfig возвращает nil (компонент не нужен)
func (c TestOptionalConfig) CacheConfig() *platformlogger.Config {
	return nil
}

// KafkaConfig возвращает nil (компонент не нужен)
func (c TestOptionalConfig) KafkaConfig() *platformlogger.Config {
	return nil
}

// GRPCConfig возвращает nil (компонент не нужен)
func (c TestOptionalConfig) GRPCConfig() *platformlogger.Config {
	return nil
}

type fakeCache struct{ closed bool }

func (f *fakeCache) Get(ctx context.Context, key string) ([]byte, error) { return nil, nil }
func (f *fakeCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return nil
}
func (f *fakeCache) Delete(ctx context.Context, key string) error { return nil }
func (f *fakeCache) Marshal(v any) ([]byte, error)                { return nil, nil }
func (f *fakeCache) Unmarshal(data []byte, v any) error           { return nil }
func (f *fakeCache) Close() error                                 { f.closed = true; return nil }

type fakeProducer struct{ closed bool }

func (p *fakeProducer) Publish(ctx context.Context, topic, key string, value []byte) error {
	return nil
}
func (p *fakeProducer) Close() error { p.closed = true; return nil }

func TestNewWithLogger(t *testing.T) {
	cfg := TestConfig{
		Logger: platformlogger.Config{
			Level:  "info",
			Format: "console",
			Output: "stdout",
		},
	}

	application, err := NewWithLogger(cfg)
	if err != nil {
		t.Fatalf("Failed to create app with logger: %v", err)
	}
	defer application.Close()

	// Проверяем, что логгер инициализирован
	if application.Logger == nil {
		t.Error("Logger should be initialized")
	}

	// Проверяем, что другие компоненты не инициализированы
	if application.Metrics != nil {
		t.Error("Metrics should not be initialized")
	}

	if application.Database != nil {
		t.Error("Database should not be initialized")
	}

	if application.Server != nil {
		t.Error("Server should not be initialized")
	}

	if application.Cache != nil {
		t.Error("Cache should not be initialized")
	}

	if application.EventPublisher != nil {
		t.Error("EventPublisher should not be initialized")
	}
}

func TestAppBuilder(t *testing.T) {
	cfg := TestConfig{
		Logger: platformlogger.Config{
			Level:  "info",
			Format: "console",
			Output: "stdout",
		},
	}

	// Тестируем Builder с минимальной конфигурацией
	application, err := NewBuilder(cfg).
		WithLogger().
		Build()
	if err != nil {
		t.Fatalf("Failed to build app: %v", err)
	}
	defer application.Close()

	// Проверяем, что логгер инициализирован
	if application.Logger == nil {
		t.Error("Logger should be initialized")
	}

	// Проверяем, что другие компоненты не инициализированы
	if application.Metrics != nil {
		t.Error("Metrics should not be initialized")
	}
}

func TestAppBuilderWithOptionalConfig(t *testing.T) {
	cfg := TestOptionalConfig{
		TestConfig: TestConfig{
			Logger: platformlogger.Config{
				Level:  "info",
				Format: "console",
				Output: "stdout",
			},
		},
	}

	// Тестируем Builder с опциональной конфигурацией
	application, err := NewBuilder(cfg).
		WithLogger().
		WithMetrics().
		WithDatabase().
		Build()
	if err != nil {
		t.Fatalf("Failed to build app with optional config: %v", err)
	}
	defer application.Close()

	// Проверяем, что логгер инициализирован
	if application.Logger == nil {
		t.Error("Logger should be initialized")
	}

	// Проверяем, что опциональные компоненты не инициализированы (так как возвращают nil)
	if application.Metrics != nil {
		t.Error("Metrics should not be initialized when config returns nil")
	}

	if application.Database != nil {
		t.Error("Database should not be initialized when config returns nil")
	}
}

func TestAppClose(t *testing.T) {
	cfg := TestConfig{
		Logger: platformlogger.Config{
			Level:  "info",
			Format: "console",
			Output: "stdout",
		},
	}

	application, err := NewWithLogger(cfg)
	if err != nil {
		t.Fatalf("Failed to create app: %v", err)
	}

	// Подготовка фейковых компонентов
	fc := &fakeCache{}
	fp := &fakeProducer{}
	application.Cache = fc
	application.EventPublisher = kafka.NewKafkaEventPublisher(fp, "test")
	application.Server = &platformserver.Server{}
	application.GRPCServer = &platformgrpc.Server{}

	var httpStopped, grpcStopped bool
	patchHTTP := monkey.PatchInstanceMethod(reflect.TypeOf(&platformserver.Server{}), "Stop", func(*platformserver.Server) error {
		httpStopped = true
		return nil
	})
	defer patchHTTP.Unpatch()

	patchGRPC := monkey.PatchInstanceMethod(reflect.TypeOf(&platformgrpc.Server{}), "Stop", func(*platformgrpc.Server, context.Context) error {
		grpcStopped = true
		return nil
	})
	defer patchGRPC.Unpatch()

	// Тестируем закрытие приложения
	err = application.Close()
	if err != nil {
		t.Errorf("Failed to close app: %v", err)
	}

	if !httpStopped {
		t.Error("HTTP server Stop was not called")
	}

	if !grpcStopped {
		t.Error("gRPC server Stop was not called")
	}

	if !fc.closed {
		t.Error("Cache Close was not called")
	}

	if !fp.closed {
		t.Error("Producer Close was not called")
	}

	// Тестируем закрытие nil приложения
	var nilApp *App
	err = nilApp.Close()
	if err != nil {
		t.Errorf("Closing nil app should not return error: %v", err)
	}
}
