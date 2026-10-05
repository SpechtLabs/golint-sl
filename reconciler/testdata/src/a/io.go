package a

import (
	"context"
	"database/sql"
	"net/http"
	"sync"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Good: client.IgnoreNotFound is the idiomatic not-found handling.
type IgnoreNotFoundReconciler struct {
	c client.Client
}

func (r *IgnoreNotFoundReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	var obj struct{}
	if err := r.c.Get(ctx, req.Name, &obj); err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}
	return reconcile.Result{}, nil
}

// feedback is not a database, whatever its method names.
type feedback struct{}

func (feedback) Exec(string) error { return nil }

type state struct {
	sync.Mutex
	mu sync.RWMutex
}

var (
	sharedState state
	globalRW    sync.RWMutex
)

// Bad: HTTP and database calls through struct fields, package variables and
// call results are recognised by their type.
type IOReconciler struct {
	db     *sql.DB
	client *http.Client
	fb     feedback
	mu     sync.Mutex
}

func (r *IOReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	_, _ = http.DefaultClient.Get("http://example.com")                          // want `reconciler should not make HTTP calls directly`
	hreq, _ := http.NewRequestWithContext(ctx, "GET", "http://example.com", nil) // want `reconciler should not make HTTP calls directly`
	_, _ = r.client.Do(hreq)                                                     // want `reconciler should not make HTTP calls directly`
	_, _ = http.PostForm("http://example.com", nil)                              // want `reconciler should not make HTTP calls directly`
	_, _ = r.db.Query("SELECT 1")                                                // want `reconciler should not access database directly`
	_, _ = r.db.ExecContext(ctx, "DELETE")                                       // want `reconciler should not access database directly`
	tx, _ := r.db.BeginTx(ctx, nil)                                              // want `reconciler should not access database directly`
	_, _ = tx.Exec("DELETE")                                                     // want `reconciler should not access database directly`
	_ = r.fb.Exec("not a database")
	feedback := feedback{}
	_ = feedback.Exec("not a database either")
	_ = r.db.Ping()
	_ = hreq.Header.Get("Accept")

	// Good: a mutex in a field of the reconciler guards its own state.
	r.mu.Lock()
	r.mu.Unlock()

	// Bad: package-level mutexes, also embedded in or a field of a
	// package-level variable, reported once per variable.
	sharedState.Lock() // want `reconciler using mutex may indicate shared state`
	sharedState.Unlock()
	sharedState.mu.RLock() // the same variable as above
	sharedState.mu.RUnlock()
	globalRW.Lock() // want `reconciler using mutex may indicate shared state`
	globalRW.Unlock()

	// Good: a local mutex is not shared.
	var local sync.Mutex
	local.Lock()
	local.Unlock()

	return reconcile.Result{}, nil
}
