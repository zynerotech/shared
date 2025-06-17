package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

var global *Logger

// Config представляет конфигурацию логгера
type Config struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"` // json или console
	Output     string `mapstructure:"output"` // stdout, stderr или путь к файлу
	TimeFormat string `mapstructure:"time_format"`
}

// Logger представляет собой обертку над zerolog.Logger
type Logger struct {
	logger zerolog.Logger
}

// New создает новый экземпляр логгера
func New(cfg Config) (*Logger, error) {
	cfg = sanitize(&cfg)
	// Настраиваем уровень логирования
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Настраиваем формат времени
	if cfg.TimeFormat == "" {
		cfg.TimeFormat = time.RFC3339
	}
	zerolog.TimeFieldFormat = cfg.TimeFormat

	// Настраиваем вывод
	var output io.Writer
	switch cfg.Output {
	case "stderr":
		output = os.Stderr
	case "stdout", "":
		output = os.Stdout
	default:
		file, err := os.OpenFile(cfg.Output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		output = file
	}

	// Настраиваем формат вывода
	if cfg.Format == "console" {
		output = zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: cfg.TimeFormat,
		}
	}

	// Создаем логгер
	logger := zerolog.New(output).With().Timestamp().Logger()

	return &Logger{
		logger: logger,
	}, nil
}

// Debug логирует сообщение с уровнем Debug
func (l *Logger) Debug() *zerolog.Event {
	return l.logger.Debug()
}

// Info логирует сообщение с уровнем Info
func (l *Logger) Info() *zerolog.Event {
	return l.logger.Info()
}

// Warn логирует сообщение с уровнем Warn
func (l *Logger) Warn() *zerolog.Event {
	return l.logger.Warn()
}

// Error логирует сообщение с уровнем Error
func (l *Logger) Error() *zerolog.Event {
	return l.logger.Error()
}

// Fatal логирует сообщение с уровнем Fatal и завершает программу
func (l *Logger) Fatal() *zerolog.Event {
	return l.logger.Fatal()
}

// With возвращает новый логгер с добавленными полями
func (l *Logger) With() zerolog.Context {
	return l.logger.With()
}

func SetGlobal(l *Logger) {
	global = l
}

func (l *Logger) Log() zerolog.Logger {
	return l.logger
}

// sanitize ensures the Config struct is populated with default values when fields are empty.
func sanitize(cfg *Config) Config {
	if cfg.Level == "" {
		cfg.Level = "info"
	}
	if cfg.Format == "" {
		cfg.Format = "json"
	}
	if cfg.Output == "" {
		cfg.Output = "stdout"
	}
	if cfg.TimeFormat == "" {
		cfg.TimeFormat = time.RFC3339
	}
	return *cfg
}
