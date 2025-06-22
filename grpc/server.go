package grpc

import (
	"context"
	"net"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_prom "github.com/grpc-ecosystem/go-grpc-prometheus"
	platformlogger "gitlab.com/zynero/shared/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

// Config represents gRPC server configuration.
type Config struct {
	Enabled               bool          `mapstructure:"enabled" default:"true"`
	Address               string        `mapstructure:"address"`
	Timeout               time.Duration `mapstructure:"timeout"`
	TLSCertFile           string        `mapstructure:"tls_cert_file"`
	TLSKeyFile            string        `mapstructure:"tls_key_file"`
	MaxConnectionAge      time.Duration `mapstructure:"max_connection_age"`
	MaxConnectionAgeGrace time.Duration `mapstructure:"max_connection_age_grace"`
	KeepAliveTime         time.Duration `mapstructure:"keep_alive_time"`
	KeepAliveTimeout      time.Duration `mapstructure:"keep_alive_timeout"`
	EnforcementMinTime    time.Duration `mapstructure:"enforcement_min_time"`
	EnforcementPermit     bool          `mapstructure:"enforcement_permit"`
}

// Server wraps a grpc.Server with additional configuration.
type Server struct {
	srv    *grpc.Server
	lis    net.Listener
	config Config
}

// NewServer creates a new gRPC server with default interceptors.
func NewServer(cfg Config, l *platformlogger.Logger, opts ...grpc.ServerOption) (*Server, error) {
	kp := keepalive.EnforcementPolicy{
		MinTime:             cfg.EnforcementMinTime,
		PermitWithoutStream: cfg.EnforcementPermit,
	}
	ka := keepalive.ServerParameters{
		Time:                  cfg.KeepAliveTime,
		Timeout:               cfg.KeepAliveTimeout,
		MaxConnectionAge:      cfg.MaxConnectionAge,
		MaxConnectionAgeGrace: cfg.MaxConnectionAgeGrace,
	}

	serverOpts := []grpc.ServerOption{
		grpc.ConnectionTimeout(cfg.Timeout),
		grpc.KeepaliveEnforcementPolicy(kp),
		grpc.KeepaliveParams(ka),
		grpc_middleware.WithUnaryServerChain(
			LoggingUnaryInterceptor(l),
			MetricsUnaryInterceptor(),
		),
		grpc_middleware.WithStreamServerChain(
			LoggingStreamInterceptor(l),
			MetricsStreamInterceptor(),
		),
	}

	if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
		creds, err := credentials.NewServerTLSFromFile(cfg.TLSCertFile, cfg.TLSKeyFile)
		if err != nil {
			return nil, err
		}
		serverOpts = append(serverOpts, grpc.Creds(creds))
	}

	for _, opt := range opts {
		if opt != nil {
			serverOpts = append(serverOpts, opt)
		}
	}

	srv := grpc.NewServer(serverOpts...)
	return &Server{srv: srv, config: cfg}, nil
}

// Start begins serving on the configured address.
func (s *Server) Start() error {
	var err error
	s.lis, err = net.Listen("tcp", s.config.Address)
	if err != nil {
		return err
	}
	grpc_prom.Register(s.srv)
	return s.srv.Serve(s.lis)
}

// Stop gracefully stops the gRPC server.
func (s *Server) Stop(ctx context.Context) error {
	stopped := make(chan struct{})
	go func() {
		s.srv.GracefulStop()
		close(stopped)
	}()
	select {
	case <-ctx.Done():
		s.srv.Stop()
		return ctx.Err()
	case <-stopped:
		return nil
	}
}

// GRPCServer exposes the underlying *grpc.Server.
func (s *Server) GRPCServer() *grpc.Server { return s.srv }
