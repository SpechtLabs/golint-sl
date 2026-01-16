---
title: statusupdate
permalink: /reference/analyzers/statusupdate
createTime: 2025/01/16 10:00:00
---

Ensures reconcilers update resource status after making changes.

## Category

Kubernetes

## What It Checks

This analyzer detects reconcilers that modify resources but don't update status, leaving users without visibility into the actual state.

## Why It Matters

Status communicates:

- Current state to users and other controllers
- Readiness for dependent operations
- Error conditions and reasons

Without status updates, users can't tell what's happening.

## Examples

### Bad: No Status Update

```go
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    obj := &myv1.MyResource{}
    if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // Create deployment
    if err := r.createDeployment(ctx, obj); err != nil {
        return ctrl.Result{}, err
    }

    // Status never updated! Users don't know deployment was created.
    return ctrl.Result{}, nil
}
```

### Good: Status Updated

```go
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    obj := &myv1.MyResource{}
    if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // Create deployment
    if err := r.createDeployment(ctx, obj); err != nil {
        obj.Status.Phase = "Failed"
        obj.Status.Message = err.Error()
        r.Status().Update(ctx, obj)
        return ctrl.Result{}, err
    }

    // Update status
    obj.Status.Phase = "Ready"
    obj.Status.DeploymentName = deploymentName(obj)
    if err := r.Status().Update(ctx, obj); err != nil {
        return ctrl.Result{}, err
    }

    return ctrl.Result{}, nil
}
```

### Good: Using Conditions

```go
import "k8s.io/apimachinery/pkg/api/meta"

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // ... get object ...

    // Set condition
    condition := metav1.Condition{
        Type:               "Ready",
        Status:             metav1.ConditionTrue,
        ObservedGeneration: obj.Generation,
        LastTransitionTime: metav1.Now(),
        Reason:             "ReconcileSucceeded",
        Message:            "All resources are ready",
    }
    meta.SetStatusCondition(&obj.Status.Conditions, condition)

    // Update status
    if err := r.Status().Update(ctx, obj); err != nil {
        return ctrl.Result{}, err
    }

    return ctrl.Result{}, nil
}
```

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  statusupdate: true  # enabled by default
```

## When to Disable

- Non-Kubernetes projects
- Controllers that don't manage custom resources

```yaml
analyzers:
  statusupdate: false
```

## Related Analyzers

- [reconciler](/reference/analyzers/reconciler) - Reconciler patterns
- [sideeffects](/reference/analyzers/sideeffects) - Side effect detection

## See Also

- [Kubernetes Patterns](/understanding/kubernetes-patterns)
