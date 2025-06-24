package logger

import (
	"sync"
	"time"
)

// GlobalConfig представляет глобальные настройки приложения для логгера
type GlobalConfig struct {
	// Основная конфигурация логгера
	Logger Config `json:"logger" yaml:"logger" mapstructure:"logger"`

	// Глобальные поля, которые будут добавлены ко всем сообщениям
	GlobalFields map[string]any `json:"global_fields" yaml:"global_fields" mapstructure:"global_fields"`

	// Информация о приложении
	Application ApplicationInfo `json:"application" yaml:"application" mapstructure:"application"`

	// Настройки для разных компонентов
	Components map[string]ComponentConfig `json:"components" yaml:"components" mapstructure:"components"`
}

// ApplicationInfo содержит информацию о приложении
type ApplicationInfo struct {
	Name        string `json:"name" yaml:"name" mapstructure:"name"`
	Version     string `json:"version" yaml:"version" mapstructure:"version"`
	Environment string `json:"environment" yaml:"environment" mapstructure:"environment"` // dev, staging, prod
	Instance    string `json:"instance" yaml:"instance" mapstructure:"instance"`          // instance ID или hostname
}

// ComponentConfig представляет настройки для конкретного компонента
type ComponentConfig struct {
	Level  string         `json:"level" yaml:"level" mapstructure:"level"`
	Fields map[string]any `json:"fields" yaml:"fields" mapstructure:"fields"`
}

var (
	globalConfig     *GlobalConfig
	globalConfigLock sync.RWMutex
	componentLoggers sync.Map // map[string]*Logger для кэширования логгеров компонентов
)

// InitGlobal инициализирует глобальные настройки приложения
func InitGlobal(cfg GlobalConfig) error {
	globalConfigLock.Lock()
	defer globalConfigLock.Unlock()

	// Применяем дефолтные значения
	cfg = sanitizeGlobalConfig(cfg)

	// Инициализируем основной логгер
	baseLogger, err := New(cfg.Logger)
	if err != nil {
		return err
	}

	// Добавляем глобальные поля приложения
	if len(cfg.GlobalFields) > 0 || cfg.Application.Name != "" {
		contextLogger := baseLogger.With()

		// Добавляем информацию о приложении
		if cfg.Application.Name != "" {
			contextLogger = contextLogger.Str("app_name", cfg.Application.Name)
		}
		if cfg.Application.Version != "" {
			contextLogger = contextLogger.Str("app_version", cfg.Application.Version)
		}
		if cfg.Application.Environment != "" {
			contextLogger = contextLogger.Str("environment", cfg.Application.Environment)
		}
		if cfg.Application.Instance != "" {
			contextLogger = contextLogger.Str("instance", cfg.Application.Instance)
		}

		// Добавляем глобальные поля
		for key, value := range cfg.GlobalFields {
			contextLogger = contextLogger.Interface(key, value)
		}

		baseLogger = contextLogger.Logger()
	}

	// Устанавливаем как глобальный логгер
	SetGlobal(baseLogger)
	globalConfig = &cfg

	// Очищаем кэш компонентов при смене конфигурации
	componentLoggers.Range(func(key, value any) bool {
		componentLoggers.Delete(key)
		return true
	})

	return nil
}

// GetGlobalConfig возвращает текущую глобальную конфигурацию
func GetGlobalConfig() *GlobalConfig {
	globalConfigLock.RLock()
	defer globalConfigLock.RUnlock()

	if globalConfig == nil {
		return nil
	}

	// Возвращаем копию для безопасности
	cfg := *globalConfig
	return &cfg
}

// UpdateGlobalFields обновляет глобальные поля во время выполнения
func UpdateGlobalFields(fields map[string]any) error {
	globalConfigLock.Lock()
	defer globalConfigLock.Unlock()

	if globalConfig == nil {
		// Разблокируем перед вызовом InitGlobal, чтобы избежать deadlock
		globalConfigLock.Unlock()
		err := InitGlobal(GlobalConfig{
			GlobalFields: fields,
		})
		globalConfigLock.Lock() // Возвращаем lock для defer
		return err
	}

	// Обновляем глобальные поля
	if globalConfig.GlobalFields == nil {
		globalConfig.GlobalFields = make(map[string]any)
	}

	for key, value := range fields {
		globalConfig.GlobalFields[key] = value
	}

	// Создаем копию конфигурации для InitGlobal
	newConfig := *globalConfig

	// Разблокируем перед вызовом InitGlobal, чтобы избежать deadlock
	globalConfigLock.Unlock()
	err := InitGlobal(newConfig)
	globalConfigLock.Lock() // Возвращаем lock для defer

	return err
}

// SetGlobalField устанавливает одно глобальное поле
func SetGlobalField(key string, value any) error {
	return UpdateGlobalFields(map[string]any{key: value})
}

// GetComponentLogger возвращает логгер для конкретного компонента с его настройками
func GetComponentLogger(componentName string) *Logger {
	// Пытаемся получить из кэша
	if cached, ok := componentLoggers.Load(componentName); ok {
		return cached.(*Logger)
	}

	globalConfigLock.RLock()
	defer globalConfigLock.RUnlock()

	baseLogger := GetGlobal()

	// Если нет глобальной конфигурации или настроек для компонента
	if globalConfig == nil {
		componentLogger := baseLogger.WithField("component", componentName)
		componentLoggers.Store(componentName, componentLogger)
		return componentLogger
	}

	componentConfig, hasComponentConfig := globalConfig.Components[componentName]

	// Создаем логгер с полем компонента
	contextLogger := baseLogger.With().Str("component", componentName)

	// Добавляем поля компонента
	if hasComponentConfig && len(componentConfig.Fields) > 0 {
		for key, value := range componentConfig.Fields {
			contextLogger = contextLogger.Interface(key, value)
		}
	}

	componentLogger := contextLogger.Logger()

	// Если у компонента свой уровень логирования, создаем отдельный экземпляр
	if hasComponentConfig && componentConfig.Level != "" {
		// Создаем новый логгер с уровнем компонента
		cfg := globalConfig.Logger
		cfg.Level = componentConfig.Level

		if newLogger, err := New(cfg); err == nil {
			// Добавляем все контекстные поля к новому логгеру
			ctx := newLogger.With().Str("component", componentName)

			// Добавляем глобальные поля приложения
			if globalConfig.Application.Name != "" {
				ctx = ctx.Str("app_name", globalConfig.Application.Name)
			}
			if globalConfig.Application.Version != "" {
				ctx = ctx.Str("app_version", globalConfig.Application.Version)
			}
			if globalConfig.Application.Environment != "" {
				ctx = ctx.Str("environment", globalConfig.Application.Environment)
			}
			if globalConfig.Application.Instance != "" {
				ctx = ctx.Str("instance", globalConfig.Application.Instance)
			}

			// Добавляем глобальные поля
			for key, value := range globalConfig.GlobalFields {
				ctx = ctx.Interface(key, value)
			}

			// Добавляем поля компонента
			for key, value := range componentConfig.Fields {
				ctx = ctx.Interface(key, value)
			}

			componentLogger = ctx.Logger()
		}
	}

	// Кэшируем логгер компонента
	componentLoggers.Store(componentName, componentLogger)
	return componentLogger
}

// Component возвращает логгер для компонента (сокращенный алиас)
func Component(name string) *Logger {
	return GetComponentLogger(name)
}

// UpdateComponentConfig обновляет конфигурацию конкретного компонента
func UpdateComponentConfig(componentName string, config ComponentConfig) error {
	globalConfigLock.Lock()
	defer globalConfigLock.Unlock()

	if globalConfig == nil {
		globalConfig = &GlobalConfig{
			Components: make(map[string]ComponentConfig),
		}
	}

	if globalConfig.Components == nil {
		globalConfig.Components = make(map[string]ComponentConfig)
	}

	globalConfig.Components[componentName] = config

	// Удаляем из кэша, чтобы пересоздать с новыми настройками
	componentLoggers.Delete(componentName)

	return nil
}

// ListComponents возвращает список всех зарегистрированных компонентов
func ListComponents() []string {
	globalConfigLock.RLock()
	defer globalConfigLock.RUnlock()

	var components []string

	// Из конфигурации
	if globalConfig != nil && globalConfig.Components != nil {
		for name := range globalConfig.Components {
			components = append(components, name)
		}
	}

	// Из кэша (компоненты, которые использовались, но не настроены)
	componentLoggers.Range(func(key, value any) bool {
		name := key.(string)
		found := false
		for _, existing := range components {
			if existing == name {
				found = true
				break
			}
		}
		if !found {
			components = append(components, name)
		}
		return true
	})

	return components
}

// GetComponentLevel возвращает уровень логирования для компонента
func GetComponentLevel(componentName string) string {
	globalConfigLock.RLock()
	defer globalConfigLock.RUnlock()

	if globalConfig != nil && globalConfig.Components != nil {
		if config, exists := globalConfig.Components[componentName]; exists && config.Level != "" {
			return config.Level
		}
	}

	// Возвращаем глобальный уровень
	if globalConfig != nil {
		return globalConfig.Logger.Level
	}

	return GetLevel()
}

// SetComponentLevel устанавливает уровень логирования для компонента
func SetComponentLevel(componentName, level string) error {
	globalConfigLock.Lock()
	defer globalConfigLock.Unlock()

	if globalConfig == nil {
		globalConfig = &GlobalConfig{
			Components: make(map[string]ComponentConfig),
		}
	}

	if globalConfig.Components == nil {
		globalConfig.Components = make(map[string]ComponentConfig)
	}

	config := globalConfig.Components[componentName]
	config.Level = level
	globalConfig.Components[componentName] = config

	// Удаляем из кэша, чтобы пересоздать с новым уровнем
	componentLoggers.Delete(componentName)

	return nil
}

// sanitizeGlobalConfig применяет дефолтные значения к глобальной конфигурации
func sanitizeGlobalConfig(cfg GlobalConfig) GlobalConfig {
	// Sanitize основной конфигурации логгера
	cfg.Logger = sanitize(&cfg.Logger)

	// Дефолтные значения для приложения
	if cfg.Application.Environment == "" {
		cfg.Application.Environment = "development"
	}

	// Инициализируем карты, если они nil
	if cfg.GlobalFields == nil {
		cfg.GlobalFields = make(map[string]any)
	}

	if cfg.Components == nil {
		cfg.Components = make(map[string]ComponentConfig)
	}

	// Добавляем timestamp как глобальное поле, если не установлено
	if _, exists := cfg.GlobalFields["startup_time"]; !exists {
		cfg.GlobalFields["startup_time"] = time.Now().Format(time.RFC3339)
	}

	return cfg
}

// ReloadGlobalConfig перезагружает глобальную конфигурацию (полезно для hot-reload)
func ReloadGlobalConfig(cfg GlobalConfig) error {
	return InitGlobal(cfg)
}
