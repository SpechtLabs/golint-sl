// Package reconciler provides analyzers specifically for Kubernetes controller/reconciler patterns.
// It ensures reconcilers follow best practices:
// - Idempotent operations
// - Proper error handling with requeue
// - No side effects outside Kubernetes API
// - Correct use of controller-runtime patterns
package reconciler

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

// Doc is the analyzer's documentation.
const Doc = `enforce Kubernetes reconciler best practices

For Reconcile methods of types named like a Reconciler, Controller or
Operator, this analyzer ensures they:
1. Return (Result, error) and take (ctx, req) parameters
2. Don't make HTTP calls directly (net/http functions and http.Client
   methods) or query a database directly (database/sql, sqlx, pgx), but
   use an injected service abstraction
3. Don't lock a package-level mutex, which shares state across
   reconciles
4. Don't sleep (use RequeueAfter) and inject a clock instead of
   calling time.Now
5. Log with a structured logger rather than fmt.Print* or log.Print*
6. Handle not-found errors after client.Get, with IsNotFound or
   client.IgnoreNotFound, so they don't requeue

These patterns ensure reliable, idempotent reconciliation.`

// Analyzer reports Kubernetes reconcilers that violate reconciler best practices.
var Analyzer = &analysis.Analyzer{
	Name:     "reconciler",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// httpCalls are the net/http functions and http.Client methods that make or
// prepare a request: Get, Post, PostForm and Head are both, Do is a method
// and the NewRequest functions are package functions.
var httpCalls = map[string]bool{
	"Get": true, "Post": true, "PostForm": true, "Head": true,
	"Do": true, "NewRequest": true, "NewRequestWithContext": true,
}

// databasePackages are the import paths, and path prefixes, of SQL clients.
var databasePackages = []string{
	"database/sql",
	"github.com/jmoiron/sqlx",
	"github.com/jackc/pgx",
}

// mutexMethods are the sync.Mutex and sync.RWMutex locking methods.
var mutexMethods = map[string]bool{
	"Lock": true, "Unlock": true, "TryLock": true,
	"RLock": true, "RUnlock": true, "TryRLock": true,
}

// ReconcileFunc tracks information about a Reconcile function. The analyzer
// doesn't use it; it is kept because it is exported.
type ReconcileFunc struct {
	Decl          *ast.FuncDecl
	ReturnsResult bool
	ReturnsError  bool
	HasRequeue    bool
}

func run(pass *analysis.Pass) (any, error) {
	reporter := nolint.NewReporter(pass)
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return
		}

		if !isReconcileFunction(fn) {
			return
		}

		// Check reconcile function signature
		checkReconcileSignature(reporter, fn)

		// Check for forbidden patterns in reconciler
		checkReconcilerBody(pass, reporter, fn)

		// Check error handling patterns
		checkErrorHandling(reporter, fn)

		// Check for proper logging
		checkLoggingPatterns(reporter, fn)
	})

	return nil, nil
}

// isReconcileFunction checks if a function is a Kubernetes reconciler
// Only returns true for the actual Reconcile method, not helper methods
func isReconcileFunction(fn *ast.FuncDecl) bool {
	if fn.Name == nil {
		return false
	}

	// Only check functions named exactly "Reconcile"
	if fn.Name.Name != "Reconcile" {
		return false
	}

	// Must have a receiver (it's a method)
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return false
	}

	// Verify the receiver type looks like a reconciler/controller/operator
	recv := fn.Recv.List[0]
	recvType := types.ExprString(recv.Type)

	patterns := []string{"Reconciler", "Controller", "Operator"}
	for _, pattern := range patterns {
		if strings.Contains(recvType, pattern) {
			return true
		}
	}

	return false
}

// checkReconcileSignature verifies the Reconcile function has correct signature
func checkReconcileSignature(reporter *nolint.Reporter, fn *ast.FuncDecl) {
	if fn.Type.Results == nil {
		reporter.Reportf(fn.Pos(), "Reconcile function must return (reconcile.Result, error)")
		return
	}

	results := fn.Type.Results.List
	if len(results) != 2 {
		reporter.Reportf(fn.Pos(), "Reconcile function must return exactly 2 values: (reconcile.Result, error)")
		return
	}

	// Check first return type is Result
	firstType := types.ExprString(results[0].Type)
	if !strings.Contains(firstType, "Result") {
		reporter.Reportf(results[0].Pos(), "first return type should be reconcile.Result, got %s", firstType)
	}

	// Check second return type is error
	secondType := types.ExprString(results[1].Type)
	if secondType != "error" {
		reporter.Reportf(results[1].Pos(), "second return type should be error, got %s", secondType)
	}

	// Check parameters
	if fn.Type.Params == nil || len(fn.Type.Params.List) < 2 {
		reporter.Reportf(fn.Pos(), "Reconcile function should have at least (ctx context.Context, req reconcile.Request) parameters")
		return
	}

	// First param should be context
	firstParam := fn.Type.Params.List[0]
	firstParamType := types.ExprString(firstParam.Type)
	if !strings.Contains(firstParamType, "Context") {
		reporter.Reportf(firstParam.Pos(), "first parameter should be context.Context")
	}
}

// checkReconcilerBody looks for forbidden patterns in reconciler body
func checkReconcilerBody(pass *analysis.Pass, reporter *nolint.Reporter, fn *ast.FuncDecl) {
	if fn.Body == nil {
		return
	}

	// Each package-level mutex is reported once, at its first use
	reportedMutexes := make(map[*types.Var]bool)

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		checkForbiddenCalls(pass, reporter, call)
		checkTimeNow(reporter, call)
		checkGlobalAccess(pass, reporter, call, reportedMutexes)

		return true
	})
}

// checkForbiddenCalls detects calls that shouldn't be in reconcilers
func checkForbiddenCalls(pass *analysis.Pass, reporter *nolint.Reporter, call *ast.CallExpr) {
	// HTTP and database calls are matched by the callee, so the client or
	// connection may come from anywhere: http.DefaultClient, r.db, a local
	// variable.
	if callee, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func); ok && callee.Pkg() != nil {
		pkgPath := callee.Pkg().Path()
		isMethod := callee.Signature().Recv() != nil

		// Methods only count on http.Client: http.Header has a Get too
		if pkgPath == "net/http" && httpCalls[callee.Name()] && (!isMethod || isHTTPClientMethod(callee)) {
			reporter.Reportf(call.Pos(),
				"reconciler should not make HTTP calls directly; use an injected HTTP client interface or service abstraction")
		}

		if isMethod && isDatabasePackage(pkgPath) && isDatabaseMethod(callee.Name()) {
			reporter.Reportf(call.Pos(),
				"reconciler should not access database directly; use repository pattern")
		}
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return
	}

	// Sleep calls (reconcilers should use requeue instead)
	if ident.Name == "time" && sel.Sel.Name == "Sleep" {
		reporter.Reportf(call.Pos(),
			"reconciler should not use time.Sleep; use Result{RequeueAfter: duration} instead")
	}
}

// isHTTPClientMethod reports whether method is declared on http.Client.
func isHTTPClientMethod(method *types.Func) bool {
	recv := types.Unalias(method.Signature().Recv().Type())
	if ptr, ok := recv.(*types.Pointer); ok {
		recv = types.Unalias(ptr.Elem())
	}
	named, ok := recv.(*types.Named)
	return ok && named.Obj().Name() == "Client"
}

// isDatabasePackage reports whether pkgPath is database/sql or a well-known
// SQL client library.
func isDatabasePackage(pkgPath string) bool {
	for _, prefix := range databasePackages {
		if pkgPath == prefix || strings.HasPrefix(pkgPath, prefix+"/") {
			return true
		}
	}
	return false
}

// isDatabaseMethod reports whether name is a query, exec, begin or prepare
// method, including their Context and sqlx variants.
func isDatabaseMethod(name string) bool {
	for _, prefix := range []string{"Query", "Exec", "Begin", "Prepare"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// checkTimeNow flags direct time.Now() usage in reconcilers
func checkTimeNow(reporter *nolint.Reporter, call *ast.CallExpr) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return
	}

	if ident.Name == "time" && sel.Sel.Name == "Now" {
		// This is informational - sometimes time.Now is needed
		// but it can make testing harder
		reporter.Reportf(call.Pos(),
			"consider injecting a clock interface for time.Now() to improve testability")
	}
}

// checkGlobalAccess looks for global variable access: locking a sync.Mutex
// or sync.RWMutex that lives in a package-level variable, directly, as a
// field of one, or embedded in one. A mutex in a field of the reconciler
// guards the reconciler's own state and is not reported.
func checkGlobalAccess(pass *analysis.Pass, reporter *nolint.Reporter, call *ast.CallExpr, reported map[*types.Var]bool) {
	callee, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
	if !ok || callee.Pkg() == nil || callee.Pkg().Path() != "sync" || !mutexMethods[callee.Name()] {
		return
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	global := packageLevelRoot(pass, sel.X)
	if global == nil || reported[global] {
		return
	}
	reported[global] = true

	// This could indicate shared state
	reporter.Reportf(call.Pos(),
		"reconciler using mutex may indicate shared state; consider using controller-runtime's built-in concurrency model")
}

// packageLevelRoot returns the package-level variable expr is rooted in
// (v, v.field.field, pkg.V, pkg.V.field), or nil.
func packageLevelRoot(pass *analysis.Pass, expr ast.Expr) *types.Var {
	for {
		switch e := ast.Unparen(expr).(type) {
		case *ast.Ident:
			return packageLevelVar(pass.TypesInfo.Uses[e])
		case *ast.SelectorExpr:
			// pkg.V is a qualified identifier; anything else is a field
			if v := packageLevelVar(pass.TypesInfo.Uses[e.Sel]); v != nil {
				return v
			}
			expr = e.X
		default:
			return nil
		}
	}
}

// packageLevelVar returns obj when it is a variable declared at package level.
func packageLevelVar(obj types.Object) *types.Var {
	v, ok := obj.(*types.Var)
	if !ok || v.IsField() || v.Pkg() == nil || v.Parent() != v.Pkg().Scope() {
		return nil
	}
	return v
}

// checkErrorHandling ensures proper error handling patterns
func checkErrorHandling(reporter *nolint.Reporter, fn *ast.FuncDecl) {
	if fn.Body == nil {
		return
	}

	// Look for apierrors.IsNotFound checks and client.IgnoreNotFound
	hasNotFoundCheck := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		hasNotFoundCheck = hasNotFoundCheck || isNotFoundCall(n)
		return !hasNotFoundCheck
	})

	// Look for client.Get calls
	hasClientGet := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		// Check if it's a client call (has context as first arg)
		if sel.Sel.Name == "Get" && len(call.Args) >= 2 {
			hasClientGet = true
		}

		return true
	})

	if hasClientGet && !hasNotFoundCheck {
		reporter.Reportf(fn.Pos(),
			"reconciler does client.Get but doesn't check for IsNotFound; not-found errors should return nil (no requeue)")
	}
}

// checkLoggingPatterns ensures structured logging is used
func checkLoggingPatterns(reporter *nolint.Reporter, fn *ast.FuncDecl) {
	if fn.Body == nil {
		return
	}

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		funcName := sel.Sel.Name

		// Check for fmt.Printf/Println in reconcilers
		if ident, ok := sel.X.(*ast.Ident); ok {
			if ident.Name == "fmt" && (funcName == "Printf" || funcName == "Println" || funcName == "Print") {
				reporter.Reportf(call.Pos(),
					"use structured logging (zap, logr) instead of fmt.Print* in reconcilers")
			}

			// Check for log.Print* (standard library logger)
			if ident.Name == "log" && strings.HasPrefix(funcName, "Print") {
				reporter.Reportf(call.Pos(),
					"use structured logging (zap, logr) instead of log.Print* in reconcilers")
			}
		}

		return true
	})
}

// ReconcilerInfo contains analysis results about a reconciler
type ReconcilerInfo struct {
	Name             string
	ForbiddenCalls   []string // never filled in; kept because it is exported
	HasProperSig     bool
	UsesRequeue      bool
	HasNotFoundCheck bool
}

// AnalyzeReconciler returns detailed information about a reconciler function
func AnalyzeReconciler(fn *ast.FuncDecl) *ReconcilerInfo {
	info := &ReconcilerInfo{
		Name: fn.Name.Name,
	}

	// Check signature
	if fn.Type.Results != nil && len(fn.Type.Results.List) == 2 {
		info.HasProperSig = true
	}

	// Analyze body
	if fn.Body != nil {
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			// Check for RequeueAfter
			if composite, ok := n.(*ast.CompositeLit); ok && setsRequeue(composite) {
				info.UsesRequeue = true
			}

			// Check for IsNotFound
			if isNotFoundCall(n) {
				info.HasNotFoundCheck = true
			}

			return true
		})
	}

	return info
}

// setsRequeue checks if a composite literal is a Result that sets Requeue or RequeueAfter
func setsRequeue(composite *ast.CompositeLit) bool {
	sel, ok := composite.Type.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Result" {
		return false
	}

	for _, elt := range composite.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}

		if ident, ok := kv.Key.(*ast.Ident); ok && (ident.Name == "RequeueAfter" || ident.Name == "Requeue") {
			return true
		}
	}

	return false
}

// isNotFoundCall checks if a node is a call that handles not-found errors:
// apierrors.IsNotFound, or client.IgnoreNotFound, which returns nil for one
func isNotFoundCall(n ast.Node) bool {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return false
	}

	var name string
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		name = fun.Sel.Name
	case *ast.Ident:
		name = fun.Name
	}
	return name == "IsNotFound" || name == "IgnoreNotFound"
}
