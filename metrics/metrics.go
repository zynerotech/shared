package metrics

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	platformlogger "gitlab.com/zynero/shared/logger"
)

// Config представляет конфигурацию метрик
type Config struct {
	Enabled     bool   `mapstructure:"enabled"`
	Path        string `mapstructure:"path"`
	Port        int    `mapstructure:"port"`
	ServiceName string `mapstructure:"service_name"`
}

// Metrics представляет собой менеджер метрик
type Metrics struct {
	config Config
	server *http.Server

	// HTTP метрики
	httpRequestsTotal    *prometheus.CounterVec
	httpRequestDuration  *prometheus.HistogramVec
	httpRequestsInFlight *prometheus.GaugeVec
}

// New создает и запускает новый экземпляр менеджера метрик
func New(cfg Config) (*Metrics, error) {
	if !cfg.Enabled {
		return &Metrics{config: cfg}, nil
	}

	m := &Metrics{
		config: cfg,
	}

	// Инициализация HTTP метрик
	m.httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_http_requests_total", cfg.ServiceName),
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	m.httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    fmt.Sprintf("%s_http_request_duration_seconds", cfg.ServiceName),
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	m.httpRequestsInFlight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: fmt.Sprintf("%s_http_requests_in_flight", cfg.ServiceName),
			Help: "Current number of HTTP requests being served",
		},
		[]string{"method", "path"},
	)

	// Запускаем HTTP-сервер для метрик
	mux := http.NewServeMux()
	mux.Handle(cfg.Path, promhttp.Handler())

	m.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: mux,
	}

	go func() {
		platformlogger.Info().Msgf("Starting metrics server on %s", m.server.Addr)
		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(fmt.Sprintf("Failed to start metrics server: %v", err))
		}
	}()

	return m, nil
}

// Stop останавливает HTTP-сервер метрик
func (m *Metrics) Stop() error {
	if !m.config.Enabled || m.server == nil {
		return nil
	}
	return m.server.Close()
}

// HTTPMiddleware возвращает middleware для сбора HTTP метрик
func (m *Metrics) HTTPMiddleware(next http.Handler) http.Handler {
	if !m.config.Enabled {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Увеличиваем счетчик текущих запросов
		m.httpRequestsInFlight.WithLabelValues(r.Method, r.URL.Path).Inc()
		defer m.httpRequestsInFlight.WithLabelValues(r.Method, r.URL.Path).Dec()

		// Создаем ResponseWriter для перехвата статуса
		rw := &responseWriter{ResponseWriter: w}

		// Вызываем следующий обработчик
		next.ServeHTTP(rw, r)

		// Записываем метрики
		duration := time.Since(start).Seconds()
		m.httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
		m.httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, fmt.Sprintf("%d", rw.status)).Inc()
	})
}

// FiberMiddleware возвращает middleware для Fiber
func (m *Metrics) FiberMiddleware() fiber.Handler {
	if !m.config.Enabled {
		return func(c *fiber.Ctx) error {
			return c.Next()
		}
	}

	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Увеличиваем счетчик текущих запросов
		m.httpRequestsInFlight.WithLabelValues(c.Method(), c.Path()).Inc()
		defer m.httpRequestsInFlight.WithLabelValues(c.Method(), c.Path()).Dec()

		// Вызываем следующий обработчик
		err := c.Next()

		// Записываем метрики
		duration := time.Since(start).Seconds()
		m.httpRequestDuration.WithLabelValues(c.Method(), c.Path()).Observe(duration)
		m.httpRequestsTotal.WithLabelValues(c.Method(), c.Path(), fmt.Sprintf("%d", c.Response().StatusCode())).Inc()

		return err
	}
}

// responseWriter перехватывает статус ответа
type responseWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader перехватывает статус ответа
func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

// Write перехватывает запись ответа
func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}
