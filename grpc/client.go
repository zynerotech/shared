package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client wraps a gRPC ClientConn with optional interceptors.
type Client struct {
	conn *grpc.ClientConn
}

// Dial creates a Client connected to the given target.
func Dial(ctx context.Context, target string, opts ...grpc.DialOption) (*Client, error) {
	opts = append([]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}, opts...)
	conn, err := grpc.DialContext(ctx, target, opts...)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn}, nil
}

// Conn returns the underlying ClientConn.
func (c *Client) Conn() *grpc.ClientConn { return c.conn }

// Close closes the underlying connection.
func (c *Client) Close() error { return c.conn.Close() }
