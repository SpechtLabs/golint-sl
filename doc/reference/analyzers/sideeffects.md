---
title: sideeffects
permalink: /reference/analyzers/sideeffects
createTime: 2025/01/16 10:00:00
---

Uses SSA analysis to detect side effects in reconcilers.

## Category

Kubernetes

## What It Checks

This analyzer uses Static Single Assignment (SSA) form to detect side effects that could break reconciler idempotency.

## Why It Matters

Reconcilers must be idempotent - running twice should produce the same result. Side effects can break this:

- Sending webhooks multiple times
- Incrementing counters
- Creating duplicate external resources

## Examples

### Bad: Non-idempotent Side Effect

```go
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    obj := &myv1.MyResource{}
    if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // This runs on every reconcile!
    if err := r.sendWebhook(obj); err != nil {
        return ctrl.Result{}, err
    }

    return ctrl.Result{}, nil
}
```

### Good: Track Side Effect State

```go
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    obj := &myv1.MyResource{}
    if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // Check if already done
    if !obj.Status.WebhookSent {
        if err := r.sendWebhook(obj); err != nil {
            return ctrl.Result{}, err
        }
        obj.Status.WebhookSent = true
        if err := r.Status().Update(ctx, obj); err != nil {
            return ctrl.Result{}, err
        }
    }

    return ctrl.Result{}, nil
}
```

### Good: Use Finalizers for Cleanup

```go
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    obj := &myv1.MyResource{}
    if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // Handle deletion
    if !obj.DeletionTimestamp.IsZero() {
        if controllerutil.ContainsFinalizer(obj, finalizerName) {
            // Cleanup side effect - runs once during deletion
            if err := r.cleanup(ctx, obj); err != nil {
                return ctrl.Result{}, err
            }
            controllerutil.RemoveFinalizer(obj, finalizerName)
            return ctrl.Result{}, r.Update(ctx, obj)
        }
        return ctrl.Result{}, nil
    }

    // Add finalizer
    if !controllerutil.ContainsFinalizer(obj, finalizerName) {
        controllerutil.AddFinalizer(obj, finalizerName)
        return ctrl.Result{}, r.Update(ctx, obj)
    }

    return r.reconcile(ctx, obj)
}
```

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  sideeffects: true  # enabled by default
```

## When to Disable

- Non-Kubernetes projects
- Performance-critical code (SSA analysis has overhead)

```yaml
analyzers:
  sideeffects: false
```

## Related Analyzers

- [reconciler](/reference/analyzers/reconciler) - Reconciler patterns
- [statusupdate](/reference/analyzers/statusupdate) - Status updates
- [dataflow](/reference/analyzers/dataflow) - General data flow analysis

## See Also

- [Kubernetes Patterns](/understanding/kubernetes-patterns)
