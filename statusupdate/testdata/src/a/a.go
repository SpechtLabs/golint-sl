// Package a holds the statusupdate cases. The analyzer is purely syntactic,
// so a tiny fake client stands in for controller-runtime.
package a

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
)

type Object struct {
	Spec   Spec
	Status ObjectStatus
}

type Spec struct{ Replicas int }

type ObjectStatus struct {
	Ready      bool
	Conditions []meta.Condition
}

type StatusWriter struct{}

func (StatusWriter) Update(ctx context.Context, obj *Object) error { return nil }

type Client struct{}

func (Client) Get(ctx context.Context, obj *Object) error    { return nil }
func (Client) Create(ctx context.Context, obj *Object) error { return nil }
func (Client) Update(ctx context.Context, obj *Object) error { return nil }
func (Client) Patch(ctx context.Context, obj *Object) error  { return nil }
func (Client) Delete(ctx context.Context, obj *Object) error { return nil }
func (Client) Status() StatusWriter                          { return StatusWriter{} }

type Result struct{}

type conditions struct{ list []meta.Condition }

func (c *conditions) SetReadyCondition(ok bool)     {}
func (c *conditions) MarkConditionUnknown(t string) {}

func helper() {}

type patchHelper struct{}

func newPatchHelper(obj *Object) *patchHelper { return &patchHelper{} }

func (*patchHelper) Patch(ctx context.Context, obj *Object) error { return nil }

// Good: mutation followed by a Status().Update().
type GoodReconciler struct{ client Client }

func (r *GoodReconciler) Reconcile(ctx context.Context, obj *Object) (Result, error) {
	if err := r.client.Create(ctx, obj); err != nil {
		return Result{}, err
	}
	return Result{}, r.client.Status().Update(ctx, obj)
}

// Bad: mutates the resource but never touches Status.
type UpdateOnlyReconciler struct{ client Client }

func (r *UpdateOnlyReconciler) Reconcile(ctx context.Context, obj *Object) (Result, error) { // want `reconciler mutates resources but doesn't update Status; use Status\(\)\.Update\(\)`
	obj.Spec.Replicas = 3
	return Result{}, r.client.Update(ctx, obj)
}

// Bad: complex logic, mutations, no Status and no conditions: both reports.
type ComplexController struct{ client Client }

func (c ComplexController) Reconcile(ctx context.Context, obj *Object) error { // want `reconciler mutates resources but doesn't update Status` `reconciler performs mutations but doesn't update Status\.Conditions`
	if obj == nil {
		return nil
	}
	if err := c.client.Get(ctx, obj); err != nil {
		return err
	}
	for i := 0; i < 3; i++ {
		helper()
	}
	return c.client.Patch(ctx, obj)
}

// Bad: Status is updated but complex logic reports no conditions.
type NoConditionsOperator struct{ client Client }

func (o *NoConditionsOperator) Reconcile(ctx context.Context, obj *Object) error { // want `reconciler performs mutations but doesn't update Status\.Conditions; consider using conditions`
	switch {
	case obj == nil:
		return nil
	}
	for range 2 {
		helper()
	}
	if err := o.client.Delete(ctx, obj); err != nil {
		return err
	}
	obj.Status.Ready = true
	return o.client.Status().Update(ctx, obj)
}

// Good: complex logic with Status and meta.SetStatusCondition.
type MetaConditionReconciler struct{ client Client }

func (r *MetaConditionReconciler) Reconcile(ctx context.Context, obj *Object, ch chan int, v any) error {
	switch v.(type) {
	case string:
		helper()
	}
	select {
	case <-ch:
	default:
	}
	if obj == nil {
		return nil
	}
	if err := r.client.Create(ctx, obj); err != nil {
		return err
	}
	for i := 0; i < 2; i++ {
		helper()
	}
	if obj.Spec.Replicas > 1 {
		helper()
	}
	meta.SetStatusCondition(&obj.Status.Conditions, meta.Condition{Type: "Ready"})
	return r.client.Status().Update(ctx, obj)
}

// Good: other meta helpers with "Condition" in their name count as condition updates.
type MetaRemoveReconciler struct{ client Client }

func (r *MetaRemoveReconciler) Reconcile(ctx context.Context, obj *Object) error {
	if obj == nil {
		return nil
	}
	if obj.Spec.Replicas == 0 {
		return nil
	}
	if meta.IsStatusConditionTrue(obj.Status.Conditions, "Ready") {
		return nil
	}
	meta.RemoveStatusCondition(&obj.Status.Conditions, "Stale")
	return r.client.Status().Update(ctx, obj)
}

// Good: a project helper named like a condition setter.
type HelperConditionReconciler struct {
	client Client
	conds  conditions
}

func (r *HelperConditionReconciler) Reconcile(ctx context.Context, obj *Object) error {
	if obj == nil {
		return nil
	}
	if obj.Spec.Replicas == 0 {
		return nil
	}
	if err := r.client.Update(ctx, obj); err != nil {
		return err
	}
	r.conds.SetReadyCondition(true)
	return r.client.Status().Update(ctx, obj)
}

// Good: a deferred patch helper (cluster-api style) persists spec and
// status; the reconciler assigns Status and its Conditions directly.
type AssignStatusReconciler struct{ client Client }

func (r *AssignStatusReconciler) Reconcile(ctx context.Context, obj *Object) error {
	helper := newPatchHelper(obj)
	defer func() { _ = helper.Patch(ctx, obj) }()

	status := ObjectStatus{}
	if obj == nil {
		return nil
	}
	if obj.Spec.Replicas == 0 {
		return nil
	}
	if obj.Spec.Replicas > 10 {
		obj.Spec.Replicas = 10
	}
	status.Conditions = nil
	obj.Status = status
	return nil
}

// Good: no mutation at all.
type ReadOnlyReconciler struct{ client Client }

func (r *ReadOnlyReconciler) Reconcile(ctx context.Context, obj *Object) error {
	return r.client.Get(ctx, obj)
}

// Good: simple logic (fewer than three branches) does not need conditions.
type SimpleReconciler struct{ client Client }

func (r *SimpleReconciler) Reconcile(ctx context.Context, obj *Object) error {
	if err := r.client.Create(ctx, obj); err != nil {
		return err
	}
	return r.client.Status().Update(ctx, obj)
}

// Good: receiver type does not look like a reconciler.
type Syncer struct{ client Client }

func (s *Syncer) Reconcile(ctx context.Context, obj *Object) error {
	return s.client.Update(ctx, obj)
}

// Good: methods on a reconciler that are not named Reconcile.
func (r *UpdateOnlyReconciler) cleanup(ctx context.Context, obj *Object) error {
	return r.client.Delete(ctx, obj)
}

// Good: a plain function named Reconcile has no receiver.
func Reconcile(ctx context.Context, c Client, obj *Object) error {
	return c.Update(ctx, obj)
}

// Good: suppressed with a nolint directive.
type SuppressedReconciler struct{ client Client }

//nolint:statusupdate
func (r *SuppressedReconciler) Reconcile(ctx context.Context, obj *Object) error {
	return r.client.Update(ctx, obj)
}
