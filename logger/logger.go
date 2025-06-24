package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

var global *Logger

// Config представляет конфигурацию логгера
type Config struct {
	Level      string `mapstructure:"level" json:"level" yaml:"level"`
	Format     string `mapstructure:"format" json:"format" yaml:"format"` // json или console
	Output     string `mapstructure:"output" json:"output" yaml:"output"` // stdout, stderr или путь к файлу
	TimeFormat string `mapstructure:"time_format" json:"time_format" yaml:"time_format"`
	CallerInfo bool   `mapstructure:"caller_info" json:"caller_info" yaml:"caller_info"` // добавлять информацию о вызывающем коде
}

// Logger представляет собой обертку над zerolog.Logger
type Logger struct {
	logger zerolog.Logger
}

// Event представляет событие логирования
type Event struct {
	event *zerolog.Event
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

	// Создаем базовый логгер
	logger := zerolog.New(output).With().Timestamp()

	// Добавляем информацию о вызывающем коде, если требуется
	if cfg.CallerInfo {
		logger = logger.Caller()
	}

	return &Logger{
		logger: logger.Logger(),
	}, nil
}

// SetGlobal устанавливает глобальный логгер
func SetGlobal(l *Logger) {
	global = l
}

// GetGlobal возвращает глобальный логгер
func GetGlobal() *Logger {
	if global == nil {
		// Создаем дефолтный логгер, если глобальный не установлен
		l, _ := New(Config{})
		SetGlobal(l)
	}
	return global
}

// Init инициализирует глобальный логгер с конфигурацией
func Init(cfg Config) error {
	l, err := New(cfg)
	if err != nil {
		return err
	}
	SetGlobal(l)
	return nil
}

// Level Methods для Logger

// Debug создает событие с уровнем Debug
func (l *Logger) Debug() *Event {
	return &Event{event: l.logger.Debug()}
}

// Info создает событие с уровнем Info
func (l *Logger) Info() *Event {
	return &Event{event: l.logger.Info()}
}

// Warn создает событие с уровнем Warn
func (l *Logger) Warn() *Event {
	return &Event{event: l.logger.Warn()}
}

// Error создает событие с уровнем Error
func (l *Logger) Error() *Event {
	return &Event{event: l.logger.Error()}
}

// Fatal создает событие с уровнем Fatal и завершает программу
func (l *Logger) Fatal() *Event {
	return &Event{event: l.logger.Fatal()}
}

// Panic создает событие с уровнем Panic и вызывает панику
func (l *Logger) Panic() *Event {
	return &Event{event: l.logger.Panic()}
}

// Trace создает событие с уровнем Trace
func (l *Logger) Trace() *Event {
	return &Event{event: l.logger.Trace()}
}

// Level Check Methods

// GetLevel возвращает текущий уровень логирования
func (l *Logger) GetLevel() zerolog.Level {
	return l.logger.GetLevel()
}

// Printf-style Methods для Logger

// Debugf логирует форматированное сообщение с уровнем Debug
func (l *Logger) Debugf(format string, v ...any) {
	l.logger.Debug().Msgf(format, v...)
}

// Infof логирует форматированное сообщение с уровнем Info
func (l *Logger) Infof(format string, v ...any) {
	l.logger.Info().Msgf(format, v...)
}

// Warnf логирует форматированное сообщение с уровнем Warn
func (l *Logger) Warnf(format string, v ...any) {
	l.logger.Warn().Msgf(format, v...)
}

// Errorf логирует форматированное сообщение с уровнем Error
func (l *Logger) Errorf(format string, v ...any) {
	l.logger.Error().Msgf(format, v...)
}

// Fatalf логирует форматированное сообщение с уровнем Fatal и завершает программу
func (l *Logger) Fatalf(format string, v ...any) {
	l.logger.Fatal().Msgf(format, v...)
}

// Panicf логирует форматированное сообщение с уровнем Panic и вызывает панику
func (l *Logger) Panicf(format string, v ...any) {
	l.logger.Panic().Msgf(format, v...)
}

// Context Methods

// With возвращает новый логгер с добавленными полями
func (l *Logger) With() *Context {
	return &Context{ctx: l.logger.With()}
}

// WithContext создает новый логгер с контекстом
func (l *Logger) WithContext(ctx context.Context) *Logger {
	return &Logger{logger: l.logger.With().Ctx(ctx).Logger()}
}

// WithFields создает новый логгер с несколькими полями
func (l *Logger) WithFields(fields map[string]any) *Logger {
	ctx := l.logger.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return &Logger{logger: ctx.Logger()}
}

// WithField создает новый логгер с одним полем
func (l *Logger) WithField(key string, value any) *Logger {
	return &Logger{logger: l.logger.With().Interface(key, value).Logger()}
}

// WithError создает новый логгер с полем error
func (l *Logger) WithError(err error) *Logger {
	return &Logger{logger: l.logger.With().Err(err).Logger()}
}

// Raw возвращает базовый zerolog.Logger для расширенного использования
func (l *Logger) Raw() zerolog.Logger {
	return l.logger
}

// Context представляет контекст для создания логгера с полями
type Context struct {
	ctx zerolog.Context
}

// Str добавляет строковое поле
func (c *Context) Str(key, val string) *Context {
	c.ctx = c.ctx.Str(key, val)
	return c
}

// Int добавляет целочисленное поле
func (c *Context) Int(key string, val int) *Context {
	c.ctx = c.ctx.Int(key, val)
	return c
}

// Int64 добавляет поле типа int64
func (c *Context) Int64(key string, val int64) *Context {
	c.ctx = c.ctx.Int64(key, val)
	return c
}

// Float64 добавляет поле типа float64
func (c *Context) Float64(key string, val float64) *Context {
	c.ctx = c.ctx.Float64(key, val)
	return c
}

// Bool добавляет булево поле
func (c *Context) Bool(key string, val bool) *Context {
	c.ctx = c.ctx.Bool(key, val)
	return c
}

// Time добавляет поле времени
func (c *Context) Time(key string, val time.Time) *Context {
	c.ctx = c.ctx.Time(key, val)
	return c
}

// Dur добавляет поле длительности
func (c *Context) Dur(key string, val time.Duration) *Context {
	c.ctx = c.ctx.Dur(key, val)
	return c
}

// Interface добавляет поле с любым типом
func (c *Context) Interface(key string, val any) *Context {
	c.ctx = c.ctx.Interface(key, val)
	return c
}

// Err добавляет поле ошибки
func (c *Context) Err(err error) *Context {
	c.ctx = c.ctx.Err(err)
	return c
}

// Logger создает логгер с накопленными полями
func (c *Context) Logger() *Logger {
	return &Logger{logger: c.ctx.Logger()}
}

// Event Methods

// Msg завершает событие логирования с сообщением
func (e *Event) Msg(msg string) {
	if e.event != nil {
		e.event.Msg(msg)
	}
}

// Msgf завершает событие логирования с форматированным сообщением
func (e *Event) Msgf(format string, v ...any) {
	if e.event != nil {
		e.event.Msgf(format, v...)
	}
}

// Send завершает событие логирования без сообщения
func (e *Event) Send() {
	if e.event != nil {
		e.event.Send()
	}
}

// Str добавляет строковое поле к событию
func (e *Event) Str(key, val string) *Event {
	if e.event != nil {
		e.event.Str(key, val)
	}
	return e
}

// Int добавляет целочисленное поле к событию
func (e *Event) Int(key string, val int) *Event {
	if e.event != nil {
		e.event.Int(key, val)
	}
	return e
}

// Int64 добавляет поле типа int64 к событию
func (e *Event) Int64(key string, val int64) *Event {
	if e.event != nil {
		e.event.Int64(key, val)
	}
	return e
}

// Float64 добавляет поле типа float64 к событию
func (e *Event) Float64(key string, val float64) *Event {
	if e.event != nil {
		e.event.Float64(key, val)
	}
	return e
}

// Bool добавляет булево поле к событию
func (e *Event) Bool(key string, val bool) *Event {
	if e.event != nil {
		e.event.Bool(key, val)
	}
	return e
}

// Time добавляет поле времени к событию
func (e *Event) Time(key string, val time.Time) *Event {
	if e.event != nil {
		e.event.Time(key, val)
	}
	return e
}

// Dur добавляет поле длительности к событию
func (e *Event) Dur(key string, val time.Duration) *Event {
	if e.event != nil {
		e.event.Dur(key, val)
	}
	return e
}

// Interface добавляет поле с любым типом к событию
func (e *Event) Interface(key string, val any) *Event {
	if e.event != nil {
		e.event.Interface(key, val)
	}
	return e
}

// Err добавляет поле ошибки к событию
func (e *Event) Err(err error) *Event {
	if e.event != nil {
		e.event.Err(err)
	}
	return e
}

// Global Functions - удобные функции для использования глобального логгера

// Debug создает событие Debug с глобальным логгером
func Debug() *Event {
	return GetGlobal().Debug()
}

// Info создает событие Info с глобальным логгером
func Info() *Event {
	return GetGlobal().Info()
}

// Warn создает событие Warn с глобальным логгером
func Warn() *Event {
	return GetGlobal().Warn()
}

// Error создает событие Error с глобальным логгером
func Error() *Event {
	return GetGlobal().Error()
}

// Fatal создает событие Fatal с глобальным логгером
func Fatal() *Event {
	return GetGlobal().Fatal()
}

// Panic создает событие Panic с глобальным логгером
func Panic() *Event {
	return GetGlobal().Panic()
}

// Trace создает событие Trace с глобальным логгером
func Trace() *Event {
	return GetGlobal().Trace()
}

// Printf-style Global Functions

// Debugf логирует форматированное сообщение Debug с глобальным логгером
func Debugf(format string, v ...any) {
	GetGlobal().Debugf(format, v...)
}

// Infof логирует форматированное сообщение Info с глобальным логгером
func Infof(format string, v ...any) {
	GetGlobal().Infof(format, v...)
}

// Warnf логирует форматированное сообщение Warn с глобальным логгером
func Warnf(format string, v ...any) {
	GetGlobal().Warnf(format, v...)
}

// Errorf логирует форматированное сообщение Error с глобальным логгером
func Errorf(format string, v ...any) {
	GetGlobal().Errorf(format, v...)
}

// Fatalf логирует форматированное сообщение Fatal с глобальным логгером
func Fatalf(format string, v ...any) {
	GetGlobal().Fatalf(format, v...)
}

// Panicf логирует форматированное сообщение Panic с глобальным логгером
func Panicf(format string, v ...any) {
	GetGlobal().Panicf(format, v...)
}

// Global Context Functions

// With возвращает контекст для создания логгера с полями
func With() *Context {
	return GetGlobal().With()
}

// WithFields создает новый логгер с несколькими полями
func WithFields(fields map[string]any) *Logger {
	return GetGlobal().WithFields(fields)
}

// WithField создает новый логгер с одним полем
func WithField(key string, value any) *Logger {
	return GetGlobal().WithField(key, value)
}

// WithError создает новый логгер с полем error
func WithError(err error) *Logger {
	return GetGlobal().WithError(err)
}

// WithContext создает новый логгер с контекстом
func WithContext(ctx context.Context) *Logger {
	return GetGlobal().WithContext(ctx)
}

// Utility Functions

// SetLevel устанавливает глобальный уровень логирования
func SetLevel(level string) error {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}
	zerolog.SetGlobalLevel(lvl)
	return nil
}

// GetLevel возвращает текущий глобальный уровень логирования
func GetLevel() string {
	return GetGlobal().GetLevel().String()
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
