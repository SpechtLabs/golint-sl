// Package grpc is a minimal stub of google.golang.org/grpc.
package grpc

type ClientConn struct{}

func Dial(target string) (*ClientConn, error) { return &ClientConn{}, nil }
