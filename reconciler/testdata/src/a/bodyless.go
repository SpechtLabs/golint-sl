package a

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Good: a Reconcile without a body (implemented in assembly) has its
// signature checked and its body checks skipped.
type AsmReconciler struct{}

func (r *AsmReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error)
