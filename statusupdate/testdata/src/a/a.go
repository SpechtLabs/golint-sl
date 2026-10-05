// Package a holds the statusupdate cases. A tiny fake client with a Status
// method stands in for controller-runtime.
package a

import (
	"context"
	"kubeutil"

	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/cluster-api/util/patch"
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
func (StatusWriter) Patch(ctx context.Context, obj *Object) error  { return nil }

type Client struct{}

func (Client) Get(ctx context.Context, obj *Object) error    { return nil }
func (Client) Create(ctx context.Context, obj *Object) error { return nil }
func (Client) Update(ctx context.Context, obj *Object) error { return nil }
func (Client) Patch(ctx context.Context, obj *Object) error  { return nil }
func (Client) Delete(ctx context.Context, obj *Object) error { return nil }
func (Client) Status() StatusWriter                          { return StatusWriter{} }

func (Client) SubResource(name string) StatusWriter { return StatusWriter{} }

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

// Bad: Status fields assigned but persisted with Update on the object, which
// ignores the status subresource.
type StatusThenUpdateReconciler struct{ client Client }

func (r *StatusThenUpdateReconciler) Reconcile(ctx context.Context, obj *Object) error { // want `reconciler mutates resources but doesn't update Status`
	obj.Status.Ready = true
	return r.client.Update(ctx, obj)
}

// Bad: the whole Status replaced, then a Patch through an embedded client,
// whose promoted Status method marks it as a client rather than a patch helper.
type StatusThenPatchReconciler struct{ Client }

func (r *StatusThenPatchReconciler) Reconcile(ctx context.Context, obj *Object) error { // want `reconciler mutates resources but doesn't update Status`
	obj.Status = ObjectStatus{Ready: true}
	return r.Patch(ctx, obj)
}

// Bad: Status() called, but the write goes to the object.
type StatusReadReconciler struct{ client Client }

func (r *StatusReadReconciler) Reconcile(ctx context.Context, obj *Object) error { // want `reconciler mutates resources but doesn't update Status`
	_ = r.client.Status()
	obj.Status.Ready = true
	return r.client.Update(ctx, obj)
}

// Good: the status persisted with Status().Patch().
type StatusPatchReconciler struct{ client Client }

func (r *StatusPatchReconciler) Reconcile(ctx context.Context, obj *Object) error {
	if err := r.client.Update(ctx, obj); err != nil {
		return err
	}
	obj.Status.Ready = true
	return r.client.Status().Patch(ctx, obj)
}

// Good: the status persisted through SubResource("status").
type SubResourceReconciler struct{ client Client }

func (r *SubResourceReconciler) Reconcile(ctx context.Context, obj *Object) error {
	if err := r.client.Create(ctx, obj); err != nil {
		return err
	}
	obj.Status.Ready = true
	return r.client.SubResource("status").Update(ctx, obj)
}

// Good: cluster-api's patch helper persists spec and status together.
type ClusterAPIReconciler struct{ client Client }

func (r *ClusterAPIReconciler) Reconcile(ctx context.Context, obj *Object) error {
	helper, err := patch.NewHelper(obj, r.client)
	if err != nil {
		return err
	}
	obj.Status.Ready = true
	return helper.Patch(ctx, obj)
}

// Bad: a package-level Patch function is a mutation, not a patch helper.
type PackagePatchReconciler struct{}

func (r *PackagePatchReconciler) Reconcile(ctx context.Context, obj *Object) error { // want `reconciler mutates resources but doesn't update Status`
	obj.Status.Ready = true
	return kubeutil.Patch(ctx, obj)
}

func writerFor(obj *Object) StatusWriter { return StatusWriter{} }

func (Client) Scale() StatusWriter { return StatusWriter{} }

// Bad: writers for other subresources, or from elsewhere, don't persist the
// status.
type OtherWriterReconciler struct{ client Client }

func (r *OtherWriterReconciler) Reconcile(ctx context.Context, obj *Object) error { // want `reconciler mutates resources but doesn't update Status`
	obj.Status.Ready = true
	if err := r.client.SubResource("scale").Update(ctx, obj); err != nil {
		return err
	}
	if err := r.client.Scale().Update(ctx, obj); err != nil {
		return err
	}
	return writerFor(obj).Update(ctx, obj)
}

// Good: the status persisted by a helper method of the reconciler.
type StatusHelperReconciler struct{ client Client }

func (r *StatusHelperReconciler) Reconcile(ctx context.Context, obj *Object) error {
	if err := r.client.Update(ctx, obj); err != nil {
		return err
	}
	obj.Status.Ready = true
	return r.updateStatus(ctx, obj)
}

func (r *StatusHelperReconciler) updateStatus(ctx context.Context, obj *Object) error {
	return persistStatus(ctx, r.client, obj)
}

func persistStatus(ctx context.Context, c Client, obj *Object) error {
	return c.Status().Patch(ctx, obj)
}

// Bad: helpers that never write the status, a recursive one among them.
type RetryReconciler struct {
	client Client
	hook   func()
}

func (r *RetryReconciler) Reconcile(ctx context.Context, obj *Object) error { // want `reconciler mutates resources but doesn't update Status`
	obj.Status.Ready = true
	r.hook()
	externalSync(obj)
	if err := r.retry(ctx, obj, 3); err != nil {
		return err
	}
	return r.client.Update(ctx, obj)
}

func (r *RetryReconciler) retry(ctx context.Context, obj *Object, n int) error {
	if err := r.client.Update(ctx, obj); err != nil && n > 0 {
		return r.retry(ctx, obj, n-1)
	}
	return nil
}

// Good: the status writer stored in a variable before the write.
type WriterVarReconciler struct{ client Client }

func (r *WriterVarReconciler) Reconcile(ctx context.Context, obj *Object) error {
	sw := r.client.Status()
	if err := r.client.Update(ctx, obj); err != nil {
		return err
	}
	obj.Status.Ready = true
	return sw.Update(ctx, obj)
}

// Good: a declared variable holding SubResource("status"), with a Patch.
type WriterDeclReconciler struct{ client Client }

func (r *WriterDeclReconciler) Reconcile(ctx context.Context, obj *Object) error {
	var sw, scale = r.client.SubResource("status"), r.client.Scale()
	if err := scale.Update(ctx, obj); err != nil {
		return err
	}
	obj.Status.Ready = true
	return sw.Patch(ctx, obj)
}

// Bad: a variable holding the writer of another subresource, and one set
// from a multi-value call.
type ScaleVarReconciler struct{ client Client }

func (r *ScaleVarReconciler) Reconcile(ctx context.Context, obj *Object) error { // want `reconciler mutates resources but doesn't update Status`
	sw := r.client.SubResource("scale")
	obj.Status.Ready = true
	if err := sw.Update(ctx, obj); err != nil {
		return err
	}
	w, ok := writerPair()
	if !ok {
		return nil
	}
	return w.Update(ctx, obj)
}

func writerPair() (StatusWriter, bool) { return StatusWriter{}, true }
