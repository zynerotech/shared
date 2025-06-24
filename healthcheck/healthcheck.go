package healthcheck

import (
	"errors"
	"fmt"
	platformlogger "gitlab.com/zynero/shared/logger"
	"net/http"
)

// Config представляет конфигурацию healthcheck
type Config struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
	Port    int    `mapstructure:"port"`
}

// Healthcheck представляет менеджер проверок здоровья
type Healthcheck struct {
	config Config
	server *http.Server
}

// New создает экземпляр health-check сервера
func New(cfg Config) (*Healthcheck, error) {
	if !cfg.Enabled {
		return &Healthcheck{config: cfg}, nil
	}

	h := &Healthcheck{
		config: cfg,
	}

	mux := http.NewServeMux()
	mux.HandleFunc(cfg.Path, h.handleHealthcheck)

	h.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: mux,
	}

	go func() {
		platformlogger.Info().Msgf("Starting healthcheck server on %s", h.server.Addr)
		if err := h.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(fmt.Sprintf("Failed to start healthcheck server: %v", err))
		}
	}()

	return h, nil
}

// Stop останавливает HTTP-сервер проверок здоровья
func (h *Healthcheck) Stop() error {
	if !h.config.Enabled || h.server == nil {
		return nil
	}
	return h.server.Close()
}

// handleHealthcheck обрабатывает запрос на проверку здоровья
func (h *Healthcheck) handleHealthcheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
