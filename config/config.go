package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Custom error types for better error handling
var (
	ErrConfigNotFound   = errors.New("config file not found")
	ErrConfigInvalid    = errors.New("invalid config")
	ErrConfigValidation = errors.New("config validation failed")
	ErrConfigUnmarshal  = errors.New("failed to unmarshal config")
)

const (
	// DefaultEnv значение окружения по умолчанию
	DefaultEnv = "dev"
	// ConfigDir директория с конфигурационными файлами
	ConfigDir = "configs"
)

// Configurable определяет интерфейс для любой конфигурации
type Configurable interface {
	Validate() error
}

// Loader предоставляет функциональность для загрузки конфигурации
type Loader struct {
	viper *viper.Viper
}

// getEnv возвращает текущее окружение
func getEnv() string {
	if env := os.Getenv("APP_ENV"); env != "" {
		return env
	}
	return DefaultEnv
}

// getConfigPath возвращает путь к конфигурационному файлу
func getConfigPath() string {
	env := getEnv()
	return filepath.Join(ConfigDir, fmt.Sprintf("%s.yaml", env))
}

// NewLoader создает новый загрузчик конфигурации
func NewLoader(configPath string) *Loader {
	v := viper.New()

	// Если путь не указан, используем путь по умолчанию
	if configPath == "" {
		configPath = getConfigPath()
	}

	v.SetConfigFile(configPath)
	v.AutomaticEnv()
	v.SetEnvPrefix("APP")

	return &Loader{
		viper: v,
	}
}

// Load загружает конфигурацию из файла в переданную структуру
func (l *Loader) Load(cfg Configurable) error {
	// Чтение файла конфига
	if err := l.viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return fmt.Errorf("%w: %v", ErrConfigNotFound, err)
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := l.viper.UnmarshalExact(cfg); err != nil {
		return fmt.Errorf("%w: %v", ErrConfigUnmarshal, err)
	}

	// Проверка конфигурацию
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrConfigValidation, err)
	}

	return nil
}

// GetConfigPath возвращает путь к файлу конфигурации
func (l *Loader) GetConfigPath() string {
	return l.viper.ConfigFileUsed()
}

// SetConfigPath устанавливает путь к файлу конфигурации
func (l *Loader) SetConfigPath(path string) {
	l.viper.SetConfigFile(path)
}

// GetConfigDir возвращает директорию с конфигурацией
func (l *Loader) GetConfigDir() string {
	return filepath.Dir(l.viper.ConfigFileUsed())
}

// Load загружает конфигурацию из файла в переданную структуру
func Load(cfg Configurable, configPath string) error {
	loader := NewLoader(configPath)
	return loader.Load(cfg)
}

// WatchConfig запускает наблюдение за изменениями конфигурационного файла
func (l *Loader) WatchConfig() {
	l.viper.WatchConfig()
}

// OnConfigChange устанавливает callback для обработки изменений конфигурации
func (l *Loader) OnConfigChange(fn func()) {
	l.viper.OnConfigChange(func(e fsnotify.Event) {
		fn()
	})
}

// GetString возвращает строковое значение из конфигурации
func (l *Loader) GetString(key string) string {
	return l.viper.GetString(key)
}

// GetStringSlice возвращает массив строк из конфигурации
func (l *Loader) GetStringSlice(key string) []string {
	return l.viper.GetStringSlice(key)
}

// GetInt возвращает целочисленное значение из конфигурации
func (l *Loader) GetInt(key string) int {
	return l.viper.GetInt(key)
}

// GetBool возвращает булево значение из конфигурации
func (l *Loader) GetBool(key string) bool {
	return l.viper.GetBool(key)
}

// GetDuration возвращает значение длительности из конфигурации
func (l *Loader) GetDuration(key string) time.Duration {
	return l.viper.GetDuration(key)
}

// SetDefault устанавливает значение по умолчанию для ключа
func (l *Loader) SetDefault(key string, value interface{}) {
	l.viper.SetDefault(key, value)
}

// GetEnv возвращает текущее окружение
func GetEnv() string {
	return getEnv()
}
