// Package reconcile is a minimal stub of sigs.k8s.io/controller-runtime/pkg/reconcile.
package reconcile

import "time"

type Request struct{ Name string }

type Result struct {
	Requeue      bool
	RequeueAfter time.Duration
}
