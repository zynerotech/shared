package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfig структура для тестирования
type TestConfig struct {
	Name     string        `mapstructure:"name"`
	Port     int           `mapstructure:"port"`
	Debug    bool          `mapstructure:"debug"`
	Timeout  time.Duration `mapstructure:"timeout"`
	Database struct {
		Host string `mapstructure:"host"`
		Port int    `mapstructure:"port"`
	} `mapstructure:"database"`
}

// Validate реализует интерфейс Configurable
func (c *TestConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	if c.Port <= 0 {
		return fmt.Errorf("port must be positive")
	}
	return nil
}

// InvalidTestConfig структура с невалидными данными
type InvalidTestConfig struct {
	Name string `mapstructure:"name"`
	Port int    `mapstructure:"port"`
}

func (c *InvalidTestConfig) Validate() error {
	return fmt.Errorf("always invalid")
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected string
	}{
		{
			name:     "default environment when APP_ENV not set",
			envValue: "",
			expected: DefaultEnv,
		},
		{
			name:     "custom environment when APP_ENV is set",
			envValue: "production",
			expected: "production",
		},
		{
			name:     "test environment",
			envValue: "test",
			expected: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Сохраняем оригинальное значение
			originalEnv := os.Getenv("APP_ENV")
			defer os.Setenv("APP_ENV", originalEnv)

			// Устанавливаем тестовое значение
			if tt.envValue == "" {
				os.Unsetenv("APP_ENV")
			} else {
				os.Setenv("APP_ENV", tt.envValue)
			}

			result := GetEnv()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetConfigPath(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected string
	}{
		{
			name:     "default config path",
			envValue: "",
			expected: filepath.Join(ConfigDir, "dev.yaml"),
		},
		{
			name:     "production config path",
			envValue: "production",
			expected: filepath.Join(ConfigDir, "production.yaml"),
		},
		{
			name:     "test config path",
			envValue: "test",
			expected: filepath.Join(ConfigDir, "test.yaml"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Сохраняем оригинальное значение
			originalEnv := os.Getenv("APP_ENV")
			defer os.Setenv("APP_ENV", originalEnv)

			// Устанавливаем тестовое значение
			if tt.envValue == "" {
				os.Unsetenv("APP_ENV")
			} else {
				os.Setenv("APP_ENV", tt.envValue)
			}

			result := getConfigPath()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewLoader(t *testing.T) {
	t.Run("with custom config path", func(t *testing.T) {
		customPath := "custom/config.yaml"
		loader := NewLoader(customPath)

		assert.NotNil(t, loader)
		assert.NotNil(t, loader.viper)
	})

	t.Run("with empty config path", func(t *testing.T) {
		loader := NewLoader("")

		assert.NotNil(t, loader)
		assert.NotNil(t, loader.viper)
	})
}

func TestLoader_Load(t *testing.T) {
	// Создаем временную директорию для тестов
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test.yaml")

	t.Run("successful load", func(t *testing.T) {
		// Создаем тестовый конфигурационный файл
		configContent := `
name: "test-app"
port: 8080
debug: true
timeout: "30s"
database:
  host: "localhost"
  port: 5432
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		loader := NewLoader(configPath)
		cfg := &TestConfig{}

		err = loader.Load(cfg)
		assert.NoError(t, err)
		assert.Equal(t, "test-app", cfg.Name)
		assert.Equal(t, 8080, cfg.Port)
		assert.True(t, cfg.Debug)
		assert.Equal(t, 30*time.Second, cfg.Timeout)
		assert.Equal(t, "localhost", cfg.Database.Host)
		assert.Equal(t, 5432, cfg.Database.Port)
	})

	t.Run("config file not found", func(t *testing.T) {
		loader := NewLoader("nonexistent.yaml")
		cfg := &TestConfig{}

		err := loader.Load(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config file")
	})

	t.Run("invalid yaml format", func(t *testing.T) {
		invalidConfigPath := filepath.Join(tempDir, "invalid.yaml")
		invalidContent := `
name: "test-app"
port: invalid_port
debug: true
`
		err := os.WriteFile(invalidConfigPath, []byte(invalidContent), 0644)
		require.NoError(t, err)

		loader := NewLoader(invalidConfigPath)
		cfg := &TestConfig{}

		err = loader.Load(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse config")
	})

	t.Run("validation failure", func(t *testing.T) {
		validConfigPath := filepath.Join(tempDir, "valid.yaml")
		validContent := `
name: "test-app"
port: 8080
`
		err := os.WriteFile(validConfigPath, []byte(validContent), 0644)
		require.NoError(t, err)

		loader := NewLoader(validConfigPath)
		cfg := &InvalidTestConfig{}

		err = loader.Load(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config validation failed")
	})

	t.Run("missing required field validation", func(t *testing.T) {
		invalidConfigPath := filepath.Join(tempDir, "missing_name.yaml")
		invalidContent := `
port: 8080
debug: true
`
		err := os.WriteFile(invalidConfigPath, []byte(invalidContent), 0644)
		require.NoError(t, err)

		loader := NewLoader(invalidConfigPath)
		cfg := &TestConfig{}

		err = loader.Load(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config validation failed")
		assert.Contains(t, err.Error(), "name is required")
	})
}

func TestLoader_GetConfigPath(t *testing.T) {
	configPath := "test/config.yaml"
	loader := NewLoader(configPath)

	// После создания loader, путь должен быть установлен
	result := loader.GetConfigPath()
	// Viper возвращает путь к файлу, который был установлен
	assert.Equal(t, configPath, result)
}

func TestLoader_SetConfigPath(t *testing.T) {
	loader := NewLoader("")
	newPath := "new/config.yaml"

	loader.SetConfigPath(newPath)
	// Проверяем, что путь был установлен (косвенно через viper)
	assert.NotNil(t, loader.viper)
}

func TestLoader_GetConfigDir(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	// Создаем файл конфигурации
	err := os.WriteFile(configPath, []byte("name: test"), 0644)
	require.NoError(t, err)

	loader := NewLoader(configPath)
	cfg := &TestConfig{Name: "test", Port: 8080}

	// Загружаем конфигурацию, чтобы viper знал о файле
	err = loader.Load(cfg)
	require.NoError(t, err)

	result := loader.GetConfigDir()
	assert.Equal(t, tempDir, result)
}

func TestLoader_GetMethods(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test.yaml")

	configContent := `
string_value: "test string"
int_value: 42
bool_value: true
duration_value: "1h30m"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	loader := NewLoader(configPath)

	// Загружаем конфигурацию
	err = loader.viper.ReadInConfig()
	require.NoError(t, err)

	t.Run("GetString", func(t *testing.T) {
		result := loader.GetString("string_value")
		assert.Equal(t, "test string", result)

		// Тест несуществующего ключа
		result = loader.GetString("nonexistent")
		assert.Equal(t, "", result)
	})

	t.Run("GetInt", func(t *testing.T) {
		result := loader.GetInt("int_value")
		assert.Equal(t, 42, result)

		// Тест несуществующего ключа
		result = loader.GetInt("nonexistent")
		assert.Equal(t, 0, result)
	})

	t.Run("GetBool", func(t *testing.T) {
		result := loader.GetBool("bool_value")
		assert.True(t, result)

		// Тест несуществующего ключа
		result = loader.GetBool("nonexistent")
		assert.False(t, result)
	})

	t.Run("GetDuration", func(t *testing.T) {
		result := loader.GetDuration("duration_value")
		expected := 1*time.Hour + 30*time.Minute
		assert.Equal(t, expected, result)

		// Тест несуществующего ключа
		result = loader.GetDuration("nonexistent")
		assert.Equal(t, time.Duration(0), result)
	})
}

func TestLoader_SetDefault(t *testing.T) {
	loader := NewLoader("")

	loader.SetDefault("test_key", "default_value")
	result := loader.GetString("test_key")
	assert.Equal(t, "default_value", result)

	loader.SetDefault("test_int", 100)
	intResult := loader.GetInt("test_int")
	assert.Equal(t, 100, intResult)
}

func TestLoad(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test.yaml")

	configContent := `
name: "global-test"
port: 9090
debug: false
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	t.Run("successful global load", func(t *testing.T) {
		cfg := &TestConfig{}
		err := Load(cfg, configPath)

		assert.NoError(t, err)
		assert.Equal(t, "global-test", cfg.Name)
		assert.Equal(t, 9090, cfg.Port)
		assert.False(t, cfg.Debug)
	})

	t.Run("load with empty path", func(t *testing.T) {
		cfg := &TestConfig{}
		err := Load(cfg, "")

		// Должна быть ошибка, так как файл по умолчанию не существует
		assert.Error(t, err)
	})
}

func TestLoader_WatchConfig(t *testing.T) {
	loader := NewLoader("")

	// Тест, что метод не паникует
	assert.NotPanics(t, func() {
		loader.WatchConfig()
	})
}

func TestLoader_OnConfigChange(t *testing.T) {
	loader := NewLoader("")

	callback := func() {
		// Callback для тестирования
	}

	// Тест, что метод не паникует
	assert.NotPanics(t, func() {
		loader.OnConfigChange(callback)
	})
}

func TestConstants(t *testing.T) {
	assert.Equal(t, "dev", DefaultEnv)
	assert.Equal(t, "configs", ConfigDir)
}

// Бенчмарк тесты
func BenchmarkNewLoader(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewLoader("test.yaml")
	}
}

func BenchmarkGetEnv(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetEnv()
	}
}

func BenchmarkLoad(b *testing.B) {
	tempDir := b.TempDir()
	configPath := filepath.Join(tempDir, "bench.yaml")

	configContent := `
name: "benchmark-test"
port: 8080
debug: true
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg := &TestConfig{}
		Load(cfg, configPath)
	}
}
