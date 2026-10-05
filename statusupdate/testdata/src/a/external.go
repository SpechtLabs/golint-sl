package a

import "context"

// Good: a Reconcile declared without a body (implemented elsewhere, for
// example in assembly) is skipped.
type ExternalReconciler struct{}

func (r *ExternalReconciler) Reconcile(ctx context.Context, obj *Object) error

// externalSync is implemented elsewhere; its body can't be followed.
func externalSync(obj *Object)
