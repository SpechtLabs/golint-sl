// Package client is a minimal stub of sigs.k8s.io/controller-runtime/pkg/client.
package client

import "context"

type Object interface{}

type Client interface {
	Get(ctx context.Context, key string, obj Object) error
}

// IgnoreNotFound returns nil on NotFound errors.
func IgnoreNotFound(err error) error { return err }
