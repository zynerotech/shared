package app

import (
	"testing"

	platformlogger "gitlab.com/zynero/shared/logger"
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

	// Тестируем закрытие приложения
	err = application.Close()
	if err != nil {
		t.Errorf("Failed to close app: %v", err)
	}

	// Тестируем закрытие nil приложения
	var nilApp *App
	err = nilApp.Close()
	if err != nil {
		t.Errorf("Closing nil app should not return error: %v", err)
	}
}
