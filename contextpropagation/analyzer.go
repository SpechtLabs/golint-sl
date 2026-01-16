// Package contextpropagation provides an analyzer that ensures context.Context
// is properly propagated through call chains for tracing, cancellation, and timeouts.
package contextpropagation

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

const Doc = `ensure context.Context is properly propagated through call chains

This analyzer detects:
1. HTTP calls without context (http.Get vs http.NewRequestWithContext)
2. Database calls without context (db.Query vs db.QueryContext)
3. context.Background()/context.TODO() when a real context is available
4. Context parameter received but not used in function body
5. Sub-calls that accept context but aren't passed the available context

Proper context propagation is critical for:
- Request tracing (OpenTelemetry, Jaeger, etc.)
- Timeout propagation
- Cancellation propagation
- Request-scoped values (user info, request ID)`

var Analyzer = &analysis.Analyzer{
	Name:     "contextpropagation",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// packageLevelCallsWithoutContext are package-level functions that should use context variants
// These are explicit package.Function patterns that we know are problematic
var packageLevelCallsWithoutContext = map[string]string{
	// net/http package-level functions (these are the problematic ones)
	"http.Get":      "use http.NewRequestWithContext and client.Do instead",
	"http.Post":     "use http.NewRequestWithContext and client.Do instead",
	"http.PostForm": "use http.NewRequestWithContext and client.Do instead",
	"http.Head":     "use http.NewRequestWithContext and client.Do instead",

	// os/exec
	"exec.Command": "use exec.CommandContext instead",

	// gRPC (common patterns)
	"grpc.Dial": "use grpc.DialContext instead",
}

// methodsRequiringContext are method names that have Context variants
// We only flag these if the first argument is NOT a context
var methodsRequiringContext = map[string]string{
	// database/sql methods
	"Query":    "use QueryContext instead",
	"QueryRow": "use QueryRowContext instead",
	"Exec":     "use ExecContext instead",
	"Prepare":  "use PrepareContext instead",
	"Begin":    "use BeginTx instead",
}

// nonContextFunctions are functions that commonly don't need context
var exemptFunctions = map[string]bool{
	"main":          true,
	"init":          true,
	"TestMain":      true,
	"BenchmarkMain": true,
}

func run(pass *analysis.Pass) (interface{}, error) {
	reporter := nolint.NewReporter(pass)
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return
		}

		// Skip exempt functions
		if fn.Name != nil && exemptFunctions[fn.Name.Name] {
			return
		}

		// Skip test functions (they often use context.Background intentionally)
		if fn.Name != nil && strings.HasPrefix(fn.Name.Name, "Test") {
			return
		}

		// Get context parameter info
		ctxParam := getContextParam(fn)
		hasContext := ctxParam != ""

		if hasContext {
			// Check if context is used
			checkContextUsed(reporter, fn, ctxParam)

			// Check for context.Background/TODO when real context available
			checkUnnecessaryBackgroundContext(reporter, fn)

			// Check calls that should use context
			checkCallsWithoutContext(reporter, fn, ctxParam)
		}

		// Even without context param, check for problematic patterns
		checkContextAwareCalls(reporter, fn, hasContext)
	})

	return nil, nil
}

// getContextParam returns the name of the context parameter if present
func getContextParam(fn *ast.FuncDecl) string {
	if fn.Type.Params == nil {
		return ""
	}

	for _, param := range fn.Type.Params.List {
		paramType := types.ExprString(param.Type)
		if strings.Contains(paramType, "context.Context") || paramType == "Context" {
			if len(param.Names) > 0 {
				return param.Names[0].Name
			}
			return "ctx" // Anonymous context param
		}
	}

	return ""
}

// checkContextUsed verifies the context parameter is actually used AND passed to sub-calls
func checkContextUsed(reporter *nolint.Reporter, fn *ast.FuncDecl, ctxParam string) {
	if fn.Body == nil {
		return
	}

	usedInCall := false    // ctx passed as argument to a function call
	usedOtherwise := false // ctx used in any other way (select, assignment, etc.)
	hasFunctionCalls := false

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			hasFunctionCalls = true
			// Check if ctx is passed as an argument
			for _, arg := range node.Args {
				if containsIdent(arg, ctxParam) {
					usedInCall = true
					return true
				}
			}

		case *ast.Ident:
			// Check for other uses (select case, assignments, etc.)
			if node.Name == ctxParam {
				usedOtherwise = true
			}
		}
		return true
	})

	if !usedInCall && !usedOtherwise {
		if ctxParam == "_" {
			reporter.Reportf(fn.Pos(),
				"context parameter is explicitly ignored with '_'; this breaks tracing and cancellation propagation")
		} else {
			reporter.Reportf(fn.Pos(),
				"context parameter %q is received but never used; pass it to sub-calls or remove it",
				ctxParam)
		}
	} else if !usedInCall && hasFunctionCalls && !isSimpleFunction(fn) {
		// Context is used but not passed to any sub-calls
		// This might indicate missing context propagation
		if ctxParam == "_" {
			reporter.Reportf(fn.Pos(),
				"context parameter is explicitly ignored with '_'; HTTP/API calls in this function won't support tracing or cancellation")
		} else {
			reporter.Reportf(fn.Pos(),
				"context parameter %q is not passed to any sub-function calls; ensure context is propagated for tracing/cancellation",
				ctxParam)
		}
	}
}

// containsIdent checks if an expression contains an identifier with the given name
func containsIdent(expr ast.Expr, name string) bool {
	found := false
	ast.Inspect(expr, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok {
			if ident.Name == name {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// isSimpleFunction checks if a function is simple enough that not propagating context is okay
// (e.g., just returns a value, only does local computation)
func isSimpleFunction(fn *ast.FuncDecl) bool {
	if fn.Body == nil {
		return true
	}

	// Very short functions are likely simple
	if len(fn.Body.List) <= 2 {
		return true
	}

	// Functions that only have assignments and returns
	hasOnlySimpleStmts := true
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.CallExpr:
			// Has function calls, not simple
			hasOnlySimpleStmts = false
			return false
		}
		return true
	})

	return hasOnlySimpleStmts
}

// checkUnnecessaryBackgroundContext detects context.Background/TODO when context available
func checkUnnecessaryBackgroundContext(reporter *nolint.Reporter, fn *ast.FuncDecl) {
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}

		if ident.Name == "context" {
			switch sel.Sel.Name {
			case "Background":
				reporter.Reportf(call.Pos(),
					"context.Background() used when context parameter is available; use the passed context instead")
			case "TODO":
				reporter.Reportf(call.Pos(),
					"context.TODO() used when context parameter is available; use the passed context instead")
			}
		}

		return true
	})
}

// checkCallsWithoutContext checks for calls that should pass context but don't
func checkCallsWithoutContext(reporter *nolint.Reporter, fn *ast.FuncDecl, ctxParam string) {
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		// Get the function being called
		callName := getCallName(call)
		if callName == "" {
			return true
		}

		// Check package-level function calls (http.Get, exec.Command, etc.)
		for pattern, advice := range packageLevelCallsWithoutContext {
			if callName == pattern {
				reporter.Reportf(call.Pos(),
					"%s called without context; %s", callName, advice)
			}
		}

		// Check method calls that should use Context variants
		// Only flag if the first argument is NOT a context
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		methodName := sel.Sel.Name
		if advice, needsContext := methodsRequiringContext[methodName]; needsContext {
			// Check if first argument is context
			if !firstArgIsContext(call, ctxParam) {
				reporter.Reportf(call.Pos(),
					"%s() called without context as first argument; %s", methodName, advice)
			}
		}

		return true
	})
}

// firstArgIsContext checks if the first argument to a call is a context
func firstArgIsContext(call *ast.CallExpr, ctxParam string) bool {
	if len(call.Args) == 0 {
		return false
	}

	firstArg := call.Args[0]

	// Check if it's the context parameter directly
	if ident, ok := firstArg.(*ast.Ident); ok {
		if ident.Name == ctxParam || ident.Name == "ctx" {
			return true
		}
	}

	// Check for context.Background(), context.TODO(), or context.WithX()
	if call, ok := firstArg.(*ast.CallExpr); ok {
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok {
				if ident.Name == "context" {
					return true
				}
			}
		}
	}

	// Check for derived contexts like ctx.WithValue, etc.
	argStr := types.ExprString(firstArg)
	if strings.Contains(argStr, "ctx") || strings.Contains(argStr, "Context") {
		return true
	}

	return false
}

// checkContextAwareCalls checks for calls that have context-aware variants
func checkContextAwareCalls(reporter *nolint.Reporter, fn *ast.FuncDecl, hasContext bool) {
	ctxParam := getContextParam(fn)

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		callName := getCallName(call)
		if callName == "" {
			return true
		}

		// ALWAYS flag http.NewRequest - there's no good reason to use it
		// Use http.NewRequestWithContext even with context.Background() to be explicit
		if callName == "http.NewRequest" {
			reporter.Reportf(call.Pos(),
				"http.NewRequest is deprecated in favor of http.NewRequestWithContext; "+
					"always use http.NewRequestWithContext(ctx, method, url, body) for proper context propagation")
		}

		// Only check the rest if context is available
		if !hasContext {
			return true
		}

		// Check for time.Sleep when context is available
		if callName == "time.Sleep" {
			reporter.Reportf(call.Pos(),
				"time.Sleep called when context is available; use select with <-ctx.Done() and time.After() instead")
		}

		// Check method calls that should propagate context
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		methodName := sel.Sel.Name

		// Check if this is a method that has a Context variant and context isn't being passed
		if advice, needsContext := methodsRequiringContext[methodName]; needsContext {
			if !firstArgIsContext(call, ctxParam) {
				reporter.Reportf(call.Pos(),
					"%s() called without context; %s", methodName, advice)
			}
		}

		return true
	})
}

// getCallName extracts a readable name from a call expression
func getCallName(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name
	case *ast.SelectorExpr:
		base := ""
		switch x := fn.X.(type) {
		case *ast.Ident:
			base = x.Name
		case *ast.SelectorExpr:
			if ident, ok := x.X.(*ast.Ident); ok {
				base = ident.Name + "." + x.Sel.Name
			}
		case *ast.CallExpr:
			// For chained calls like client.Get().Do()
			base = getCallName(x)
		}
		if base != "" {
			return base + "." + fn.Sel.Name
		}
		return fn.Sel.Name
	}
	return ""
}

// ContextPropagationInfo contains analysis results
type ContextPropagationInfo struct {
	FunctionsWithContext    int
	FunctionsWithoutContext int
	ContextIgnored          int
	BackgroundContextUsed   int
	CallsWithoutContext     int
}

// AnalyzeContextPropagation returns information about context usage
func AnalyzeContextPropagation(pass *analysis.Pass) *ContextPropagationInfo {
	info := &ContextPropagationInfo{}
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return
		}

		if getContextParam(fn) != "" {
			info.FunctionsWithContext++
		} else {
			info.FunctionsWithoutContext++
		}
	})

	return info
}
