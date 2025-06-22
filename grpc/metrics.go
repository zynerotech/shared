package grpc

import (
	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"
)

// MetricsUnaryInterceptor provides Prometheus metrics for unary calls.
func MetricsUnaryInterceptor() grpc.UnaryServerInterceptor {
	return grpc_prometheus.UnaryServerInterceptor
}

// MetricsStreamInterceptor provides Prometheus metrics for streams.
func MetricsStreamInterceptor() grpc.StreamServerInterceptor {
	return grpc_prometheus.StreamServerInterceptor
}
