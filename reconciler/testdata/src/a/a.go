package a

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var globalMu sync.Mutex

// Good: correct signature, client.Get with an IsNotFound check, structured
// logging only, and helper calls that are not flagged.
type GoodReconciler struct {
	client client.Client
}

func (r *GoodReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	var obj struct{}
	if err := r.client.Get(ctx, req.Name, &obj); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	q := url.Values{}
	_ = q.Get("single-arg Get is not a client Get")
	_ = fmt.Sprintf("%d", len(req.Name))
	log.SetPrefix("x")
	return reconcile.Result{RequeueAfter: time.Minute}, nil
}

// Good: helper methods are not reconcilers, so their calls are not checked.
func (r *GoodReconciler) helper() {
	time.Sleep(time.Second)
}

// Good: the receiver does not look like a reconciler.
type Service struct{}

func (s *Service) Reconcile() {
	time.Sleep(time.Second)
}

// Good: plain functions are not reconcilers.
func Reconcile() {
	time.Sleep(time.Second)
}

// Bad: no results at all.
type NoResultReconciler struct{}

func (r *NoResultReconciler) Reconcile(ctx context.Context, req reconcile.Request) { // want `Reconcile function must return \(reconcile.Result, error\)`
}

// Bad: a single result.
type OneResultController struct{}

func (c OneResultController) Reconcile(ctx context.Context, req reconcile.Request) error { // want `Reconcile function must return exactly 2 values: \(reconcile.Result, error\)`
	return nil
}

// Bad: wrong result types.
type WrongTypesOperator struct{}

func (o *WrongTypesOperator) Reconcile(ctx context.Context, req reconcile.Request) (int, string) { // want `first return type should be reconcile.Result, got int` `second return type should be error, got string`
	return 0, ""
}

// Bad: too few parameters.
type FewParamsReconciler struct{}

func (r *FewParamsReconciler) Reconcile(req reconcile.Request) (reconcile.Result, error) { // want `Reconcile function should have at least \(ctx context.Context, req reconcile.Request\) parameters`
	return reconcile.Result{}, nil
}

// Bad: the first parameter is not a context.
type ParamOrderReconciler struct{}

func (r *ParamOrderReconciler) Reconcile(req reconcile.Request, ctx context.Context) (reconcile.Result, error) { // want `first parameter should be context.Context`
	return reconcile.Result{}, nil
}

// Bad: client.Get without an IsNotFound check.
type NoNotFoundReconciler struct {
	client client.Client
}

func (r *NoNotFoundReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) { // want `reconciler does client.Get but doesn't check for IsNotFound; not-found errors should return nil \(no requeue\)`
	var obj struct{}
	err := r.client.Get(ctx, req.Name, &obj)
	return reconcile.Result{}, err
}

// Bad: forbidden calls, global locks and unstructured logging.
type SideEffectReconciler struct{}

func (r *SideEffectReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	_, _ = http.Get("http://example.com")                     // want `reconciler should not make HTTP calls directly; use an injected HTTP client interface or service abstraction`
	_, _ = http.Post("http://example.com", "text/plain", nil) // want `reconciler should not make HTTP calls directly`
	_, _ = http.Head("http://example.com")                    // want `reconciler should not make HTTP calls directly`
	_, _ = http.NewRequest("GET", "http://example.com", nil)  // want `reconciler should not make HTTP calls directly`
	_ = http.StatusText(200)

	var db *sql.DB
	_, _ = db.Query("SELECT 1")   // want `reconciler should not access database directly; use repository pattern`
	_ = db.QueryRow("SELECT 1")   // want `reconciler should not access database directly`
	_, _ = db.Exec("DELETE")      // want `reconciler should not access database directly`
	_, _ = db.Begin()             // want `reconciler should not access database directly`
	_, _ = db.Prepare("SELECT 1") // want `reconciler should not access database directly`
	appDB := db
	_, _ = appDB.Exec("DELETE") // want `reconciler should not access database directly`
	_ = db.Ping()

	time.Sleep(time.Second) // want `reconciler should not use time.Sleep; use Result\{RequeueAfter: duration\} instead`
	_ = time.Now()          // want `consider injecting a clock interface for time.Now\(\) to improve testability`
	_ = time.Since(time.Time{})

	globalMu.Lock()   // want `reconciler using mutex may indicate shared state; consider using controller-runtime's built-in concurrency model`
	globalMu.Unlock() // reported once per mutex, at the Lock above

	fmt.Printf("%s\n", req.Name) // want `use structured logging \(zap, logr\) instead of fmt.Print\* in reconcilers`
	fmt.Println(req.Name)        // want `use structured logging \(zap, logr\) instead of fmt.Print\* in reconcilers`
	fmt.Print(req.Name)          // want `use structured logging \(zap, logr\) instead of fmt.Print\* in reconcilers`
	log.Printf("%s", req.Name)   // want `use structured logging \(zap, logr\) instead of log.Print\* in reconcilers`
	log.Println(req.Name)        // want `use structured logging \(zap, logr\) instead of log.Print\* in reconcilers`

	time.Sleep(time.Millisecond) //nolint:reconciler

	func() {}()
	return reconcile.Result{}, nil
}
