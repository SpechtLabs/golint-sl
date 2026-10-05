// Package patch is a stub of sigs.k8s.io/cluster-api/util/patch for testing
// the statusupdate analyzer.
package patch

import "context"

// Helper persists the changes to an object's spec and status.
type Helper struct{}

// NewHelper returns a Helper for obj.
func NewHelper(obj, client any) (*Helper, error) { return &Helper{}, nil }

// Patch persists the changes made to obj since NewHelper.
func (h *Helper) Patch(ctx context.Context, obj any) error { return nil }
