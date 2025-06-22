package grpc

import (
	"context"
	"time"

	platformlogger "gitlab.com/zynero/shared/logger"
	"google.golang.org/grpc"
)

// LoggingUnaryInterceptor returns a unary server interceptor for logging.
func LoggingUnaryInterceptor(l *platformlogger.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		if l != nil {
			l.Info().Str("method", info.FullMethod).Dur("duration", time.Since(start)).Err(err).Msg("grpc request")
		}
		return resp, err
	}
}

// LoggingStreamInterceptor returns a stream server interceptor for logging.
func LoggingStreamInterceptor(l *platformlogger.Logger) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		err := handler(srv, ss)
		if l != nil {
			l.Info().Str("method", info.FullMethod).Dur("duration", time.Since(start)).Err(err).Msg("grpc stream")
		}
		return err
	}
}
