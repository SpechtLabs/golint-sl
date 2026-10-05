// Package grpc is a stub of google.golang.org/grpc for testing the
// resourceclose analyzer.
package grpc

// ClientConn is a client connection to a gRPC server.
type ClientConn struct{}

// Close tears down the connection.
func (*ClientConn) Close() error { return nil }

// NewClient creates a client connection.
func NewClient(target string) (*ClientConn, error) { return &ClientConn{}, nil }

// Dial creates a client connection (deprecated upstream).
func Dial(target string) (*ClientConn, error) { return &ClientConn{}, nil }
