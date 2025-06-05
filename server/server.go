package server

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2/middleware/compress"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// Config представляет конфигурацию веб-сервера
type Config struct {
	Address         string        `mapstructure:"address"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// Server представляет веб-сервер на основе Fiber
type Server struct {
	app    *fiber.App
	config Config
}

// New создает новый экземпляр веб-сервера
func New(cfg Config) (*Server, error) {
	// Создаем конфигурацию Fiber
	fiberConfig := fiber.Config{
		DisableStartupMessage: true,
		ReadTimeout:           cfg.ReadTimeout,
		WriteTimeout:          cfg.WriteTimeout,
		IdleTimeout:           cfg.IdleTimeout,
		JSONEncoder: func(v any) ([]byte, error) {
			return sonic.Marshal(v)
		},
		JSONDecoder: func(data []byte, v any) error {
			return sonic.Unmarshal(data, v)
		},
	}

	// Создаем приложение Fiber
	app := fiber.New(fiberConfig)

	// Добавляем middleware
	app.Use(compress.New())
	app.Use(recover.New())

	return &Server{
		app:    app,
		config: cfg,
	}, nil
}

// Start запускает веб-сервер
func (s *Server) Start() error {
	return s.app.Listen(s.config.Address)
}

// Stop останавливает веб-сервер
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()
	return s.app.ShutdownWithContext(ctx)
}

// App возвращает экземпляр приложения Fiber
func (s *Server) App() *fiber.App {
	return s.app
}
