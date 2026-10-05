---
title: reconciler
permalink: /reference/analyzers/reconciler
createTime: 2025/01/16 10:00:00
---

Enforces Kubernetes reconciler best practices.

## Category

Kubernetes

## What It Checks

This analyzer checks the `Reconcile` methods of types named like a `Reconciler`, `Controller` or `Operator`:

- the signature is `(ctx context.Context, req reconcile.Request) (reconcile.Result, error)`
- a method that calls `client.Get` also handles not-found errors, with `apierrors.IsNotFound` or `client.IgnoreNotFound`, so a deleted object doesn't requeue
- no direct HTTP calls (`net/http` functions and `http.Client` methods, wherever the client comes from) or database calls (`database/sql`, `sqlx`, `pgx`)
- no locking of a package-level mutex, which shares state across concurrent reconciles; a mutex in a field of the reconciler is fine
- no `time.Sleep` (use `RequeueAfter`), and a hint to inject a clock for `time.Now`
- structured logging instead of `fmt.Print*` and `log.Print*`

## Why It Matters

Reconcilers have subtle requirements. Incorrect patterns cause:

- Infinite reconcile loops
- Missed updates
- Resource leaks
- Controller crashes

## Examples

### Bad: Not Handling NotFound

```go
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    obj := &myv1.MyResource{}
    if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
        return ctrl.Result{}, err  // Error logged for normal deletions!
    }
    // ...
}
```

### Good: Ignore NotFound

```go
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    obj := &myv1.MyResource{}
    if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }
    // ...
}
```

### Bad: Tight Requeue Loop

```go
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // ... work ...
    return ctrl.Result{Requeue: true}, nil  // Tight loop, no backoff!
}
```

### Good: Explicit Backoff

```go
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // ... work ...
    return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}
```

### Good: Let Controller-Runtime Handle Backoff

```go
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // ... work ...
    if err != nil {
        return ctrl.Result{}, err  // Exponential backoff on error
    }
    return ctrl.Result{}, nil  // No requeue if successful
}
```

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  reconciler: true  # enabled by default
```

## When to Disable

- Non-Kubernetes projects
- Projects not using controller-runtime

```yaml
analyzers:
  reconciler: false
```

## Related Analyzers

- [statusupdate](/reference/analyzers/statusupdate) - Status update requirements
- [sideeffects](/reference/analyzers/sideeffects) - Side effect detection

## See Also

- [Kubernetes Patterns](/understanding/kubernetes-patterns)
- [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime)
