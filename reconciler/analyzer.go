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
)

const Doc = `enforce Kubernetes reconciler best practices

This analyzer ensures reconcilers:
1. Always return (Result, error) with proper requeue logic
2. Don't make external HTTP calls (use service abstraction)
3. Don't access global state
4. Use proper logging patterns with structured fields
5. Handle not-found errors correctly (don't requeue)

These patterns ensure reliable, idempotent reconciliation.`

var Analyzer = &analysis.Analyzer{
	Name:     "reconciler",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// ReconcileFunc tracks information about a Reconcile function
type ReconcileFunc struct {
	Decl          *ast.FuncDecl
	ReturnsResult bool
	ReturnsError  bool
	HasRequeue    bool
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return
		}

		if !isReconcileFunction(fn) {
			return
		}

		// Check reconcile function signature
		checkReconcileSignature(pass, fn)

		// Check for forbidden patterns in reconciler
		checkReconcilerBody(pass, fn)

		// Check error handling patterns
		checkErrorHandling(pass, fn)

		// Check for proper logging
		checkLoggingPatterns(pass, fn)
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
func checkReconcileSignature(pass *analysis.Pass, fn *ast.FuncDecl) {
	if fn.Type.Results == nil {
		pass.Reportf(fn.Pos(), "Reconcile function must return (reconcile.Result, error)")
		return
	}

	results := fn.Type.Results.List
	if len(results) != 2 {
		pass.Reportf(fn.Pos(), "Reconcile function must return exactly 2 values: (reconcile.Result, error)")
		return
	}

	// Check first return type is Result
	firstType := types.ExprString(results[0].Type)
	if !strings.Contains(firstType, "Result") {
		pass.Reportf(results[0].Pos(), "first return type should be reconcile.Result, got %s", firstType)
	}

	// Check second return type is error
	secondType := types.ExprString(results[1].Type)
	if secondType != "error" {
		pass.Reportf(results[1].Pos(), "second return type should be error, got %s", secondType)
	}

	// Check parameters
	if fn.Type.Params == nil || len(fn.Type.Params.List) < 2 {
		pass.Reportf(fn.Pos(), "Reconcile function should have at least (ctx context.Context, req reconcile.Request) parameters")
		return
	}

	// First param should be context
	firstParam := fn.Type.Params.List[0]
	firstParamType := types.ExprString(firstParam.Type)
	if !strings.Contains(firstParamType, "Context") {
		pass.Reportf(firstParam.Pos(), "first parameter should be context.Context")
	}
}

// checkReconcilerBody looks for forbidden patterns in reconciler body
func checkReconcilerBody(pass *analysis.Pass, fn *ast.FuncDecl) {
	if fn.Body == nil {
		return
	}

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		checkForbiddenCalls(pass, call)
		checkTimeNow(pass, call)
		checkGlobalAccess(pass, call)

		return true
	})
}

// checkForbiddenCalls detects calls that shouldn't be in reconcilers
func checkForbiddenCalls(pass *analysis.Pass, call *ast.CallExpr) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return
	}

	pkg := ident.Name
	funcName := sel.Sel.Name

	// HTTP client calls
	if pkg == "http" {
		httpMethods := []string{"Get", "Post", "Head", "Put", "Delete", "Do", "NewRequest"}
		for _, method := range httpMethods {
			if funcName == method {
				pass.Reportf(call.Pos(),
					"reconciler should not make HTTP calls directly; use an injected HTTP client interface or service abstraction")
			}
		}
	}

	// SQL database calls
	if pkg == "sql" || strings.Contains(pkg, "db") || strings.Contains(pkg, "DB") {
		sqlMethods := []string{"Query", "QueryRow", "Exec", "Begin", "Prepare"}
		for _, method := range sqlMethods {
			if funcName == method {
				pass.Reportf(call.Pos(),
					"reconciler should not access database directly; use repository pattern")
			}
		}
	}

	// Sleep calls (reconcilers should use requeue instead)
	if pkg == "time" && funcName == "Sleep" {
		pass.Reportf(call.Pos(),
			"reconciler should not use time.Sleep; use Result{RequeueAfter: duration} instead")
	}
}

// checkTimeNow flags direct time.Now() usage in reconcilers
func checkTimeNow(pass *analysis.Pass, call *ast.CallExpr) {
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
		pass.Reportf(call.Pos(),
			"consider injecting a clock interface for time.Now() to improve testability")
	}
}

// checkGlobalAccess looks for global variable access
func checkGlobalAccess(pass *analysis.Pass, call *ast.CallExpr) {
	// Check for sync.Mutex Lock/Unlock on package-level variables
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	if sel.Sel.Name == "Lock" || sel.Sel.Name == "Unlock" {
		// This could indicate shared state
		pass.Reportf(call.Pos(),
			"reconciler using mutex may indicate shared state; consider using controller-runtime's built-in concurrency model")
	}
}

// checkErrorHandling ensures proper error handling patterns
func checkErrorHandling(pass *analysis.Pass, fn *ast.FuncDecl) {
	if fn.Body == nil {
		return
	}

	hasNotFoundCheck := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		// Look for IsNotFound checks
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		// Check for k8serrors.IsNotFound or apierrors.IsNotFound
		if sel.Sel.Name == "IsNotFound" {
			hasNotFoundCheck = true
		}

		return true
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

		if sel.Sel.Name == "Get" {
			// Check if it's a client call (has context as first arg)
			if len(call.Args) >= 2 {
				hasClientGet = true
			}
		}

		return true
	})

	if hasClientGet && !hasNotFoundCheck {
		pass.Reportf(fn.Pos(),
			"reconciler does client.Get but doesn't check for IsNotFound; not-found errors should return nil (no requeue)")
	}
}

// checkLoggingPatterns ensures structured logging is used
func checkLoggingPatterns(pass *analysis.Pass, fn *ast.FuncDecl) {
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
				pass.Reportf(call.Pos(),
					"use structured logging (zap, logr) instead of fmt.Print* in reconcilers")
			}

			// Check for log.Print* (standard library logger)
			if ident.Name == "log" && strings.HasPrefix(funcName, "Print") {
				pass.Reportf(call.Pos(),
					"use structured logging (zap, logr) instead of log.Print* in reconcilers")
			}
		}

		return true
	})
}

// ReconcilerInfo contains analysis results about a reconciler
type ReconcilerInfo struct {
	Name             string
	HasProperSig     bool
	UsesRequeue      bool
	HasNotFoundCheck bool
	ForbiddenCalls   []string
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
			if composite, ok := n.(*ast.CompositeLit); ok {
				if sel, ok := composite.Type.(*ast.SelectorExpr); ok {
					if sel.Sel.Name == "Result" {
						for _, elt := range composite.Elts {
							if kv, ok := elt.(*ast.KeyValueExpr); ok {
								if ident, ok := kv.Key.(*ast.Ident); ok {
									if ident.Name == "RequeueAfter" || ident.Name == "Requeue" {
										info.UsesRequeue = true
									}
								}
							}
						}
					}
				}
			}

			// Check for IsNotFound
			if call, ok := n.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
					if sel.Sel.Name == "IsNotFound" {
						info.HasNotFoundCheck = true
					}
				}
			}

			return true
		})
	}

	return info
}
