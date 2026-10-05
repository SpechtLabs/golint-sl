// Package contextpropagation provides an analyzer that ensures context.Context
// is properly propagated through call chains for tracing, cancellation, and timeouts.
package contextpropagation

import (
	"go/ast"
	"go/types"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

// Doc is the contextpropagation analyzer's documentation.
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

// Analyzer reports context.Context that is not propagated through call chains.
var Analyzer = &analysis.Analyzer{
	Name:     "contextpropagation",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// httpRequestWithContextAdvice is the advice for net/http package-level
// request functions.
const httpRequestWithContextAdvice = "use http.NewRequestWithContext and client.Do instead"

// packageLevelCallsWithoutContext are package-level functions that should use context variants
// These are explicit package.Function patterns that we know are problematic
var packageLevelCallsWithoutContext = map[string]string{
	// net/http package-level functions (these are the problematic ones)
	"http.Get":      httpRequestWithContextAdvice,
	"http.Post":     httpRequestWithContextAdvice,
	"http.PostForm": httpRequestWithContextAdvice,
	"http.Head":     httpRequestWithContextAdvice,

	// os/exec
	"exec.Command": "use exec.CommandContext instead",

	// gRPC (common patterns)
	"grpc.Dial": "use grpc.DialContext instead",
}

// methodsRequiringContext maps method names to their context-aware variants.
// A call is only flagged when its receiver actually has the variant (as
// database/sql's DB, Tx and Conn do) and its first argument is not a context.
var methodsRequiringContext = map[string]string{
	// database/sql methods
	"Query":    "QueryContext",
	"QueryRow": "QueryRowContext",
	"Exec":     "ExecContext",
	"Prepare":  "PrepareContext",
	"Begin":    "BeginTx",
}

// nonContextFunctions are functions that commonly don't need context
var exemptFunctions = map[string]bool{
	"main":          true,
	"init":          true,
	"TestMain":      true,
	"BenchmarkMain": true,
}

// isMockPackage checks if the current package is a mock package
func isMockPackage(pass *analysis.Pass) bool {
	pkgPath := pass.Pkg.Path()
	pkgName := pass.Pkg.Name()
	return strings.Contains(pkgPath, "/mock") ||
		strings.HasSuffix(pkgPath, "/mock") ||
		pkgName == "mock" ||
		strings.HasPrefix(pkgName, "mock")
}

// isMockFunction checks if a function is on a mock type (receiver starts with Mock)
func isMockFunction(fn *ast.FuncDecl) bool {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return false
	}
	// Check receiver type
	recv := fn.Recv.List[0]
	typeName := ""
	switch t := recv.Type.(type) {
	case *ast.Ident:
		typeName = t.Name
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			typeName = ident.Name
		}
	}
	return strings.HasPrefix(typeName, "Mock") || strings.HasPrefix(typeName, "mock")
}

func run(pass *analysis.Pass) (any, error) {
	reporter := nolint.NewReporter(pass)
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	isMockPkg := isMockPackage(pass)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return
		}

		// Only the file's own name is looked at: the directories above it are
		// wherever the module is checked out, and a mock/ or mocks/ directory
		// inside the module is a mock package (see isMockPackage)
		fileName := filepath.Base(pass.Fset.Position(fn.Pos()).Filename)

		// Skip test files entirely
		if strings.HasSuffix(fileName, "_test.go") {
			return
		}

		// Skip mock files - mocks often intentionally ignore context
		if strings.HasSuffix(fileName, "_mock.go") || strings.HasSuffix(fileName, "_mocks.go") {
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

		// Skip mock packages and mock type methods - mocks often don't propagate context
		if isMockPkg || isMockFunction(fn) {
			return
		}

		// Get context parameter info
		ctxParam := getContextParam(fn)
		hasContext := ctxParam != ""

		if hasContext {
			// Check if context is used
			checkContextUsed(pass, reporter, fn, ctxParam)

			// Check for context.Background/TODO when real context available
			checkUnnecessaryBackgroundContext(reporter, fn)

			// Check calls that should use context
			checkCallsWithoutContext(pass, reporter, fn)
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
		if isContextTypeExpr(param.Type) {
			if len(param.Names) > 0 {
				return param.Names[0].Name
			}
			return "ctx" // Anonymous context param
		}
	}

	return ""
}

// contextMethods are methods on context.Context that represent meaningful usage
// When these are called, the context IS being used even if not passed to sub-calls
var contextMethods = map[string]bool{
	"Done":     true, // ctx.Done() for cancellation
	"Deadline": true, // ctx.Deadline() for timeout checking
	"Err":      true, // ctx.Err() for error checking
	"Value":    true, // ctx.Value() for retrieving values
}

// contextUsage records how a function body uses its context parameter.
type contextUsage struct {
	usedInCall        bool // ctx passed as argument to a function call
	usedContextMethod bool // ctx methods called (ctx.Done(), ctx.Err(), etc.)
	storedInField     bool // ctx stored in a struct field (m.ctx = ctx)
	usedOtherwise     bool // ctx used in any other way (select, assignment, etc.)
	hasFunctionCalls  bool
}

// checkContextUsed verifies the context parameter is actually used AND passed to sub-calls
func checkContextUsed(pass *analysis.Pass, reporter *nolint.Reporter, fn *ast.FuncDecl, ctxParam string) {
	if fn.Body == nil {
		return
	}

	usage := scanContextUsage(pass, fn)

	// Context is meaningfully used if:
	// 1. Passed to a sub-call, OR
	// 2. A context method is called (Done, Deadline, Err, Value), OR
	// 3. Stored in a struct field for later use
	contextMeaningfullyUsed := usage.usedInCall || usage.usedContextMethod || usage.storedInField
	makesCalls := usage.hasFunctionCalls && !isSimpleFunction(fn)

	switch {
	case ctxParam == "_" && makesCalls:
		// A blank context can't be used; say why it matters when calls are made
		reporter.Reportf(fn.Pos(),
			"context parameter is explicitly ignored with '_'; HTTP/API calls in this function won't support tracing or cancellation")
	case ctxParam == "_":
		reporter.Reportf(fn.Pos(),
			"context parameter is explicitly ignored with '_'; this breaks tracing and cancellation propagation")
	case !contextMeaningfullyUsed && !usage.usedOtherwise:
		reporter.Reportf(fn.Pos(),
			"context parameter %q is received but never used; pass it to sub-calls or remove it",
			ctxParam)
	case !contextMeaningfullyUsed && makesCalls:
		// Context is referenced but not used meaningfully (not passed to calls, no methods called)
		// This might indicate missing context propagation
		reporter.Reportf(fn.Pos(),
			"context parameter %q is not passed to any sub-function calls; ensure context is propagated for tracing/cancellation",
			ctxParam)
	}
}

// scanContextUsage walks fn's body and records how it uses the context
// parameter and the local variables it was copied into (c := ctx), matched by
// object identity. ast.Inspect visits an assignment before the statements
// after it, so one pass collects the copies.
func scanContextUsage(pass *analysis.Pass, fn *ast.FuncDecl) contextUsage {
	var usage contextUsage

	ctxVars := make(map[types.Object]bool)
	if v := contextParamObject(pass, fn); v != nil {
		ctxVars[v] = true
	}
	isCtx := func(ident *ast.Ident) bool {
		obj := pass.TypesInfo.ObjectOf(ident)
		return obj != nil && ctxVars[obj]
	}

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.AssignStmt:
			// Check if context is being stored in a field (e.g., m.ctx = ctx)
			if storesIdentInField(node, isCtx) {
				usage.storedInField = true
			}
			trackContextCopies(pass, node.Lhs, node.Rhs, isCtx, ctxVars)

		case *ast.ValueSpec:
			names := make([]ast.Expr, len(node.Names))
			for i, name := range node.Names {
				names[i] = name
			}
			trackContextCopies(pass, names, node.Values, isCtx, ctxVars)

		case *ast.CallExpr:
			usage.hasFunctionCalls = true

			// Check if this is a method call on the context (ctx.Done(), ctx.Err(), etc.)
			if isContextMethodCall(node, isCtx) {
				usage.usedContextMethod = true
				return true
			}

			// Check if ctx is passed as an argument
			for _, arg := range node.Args {
				if containsIdent(arg, isCtx) {
					usage.usedInCall = true
					return true
				}
			}

		case *ast.Ident:
			// Check for other uses (select case, assignments, etc.)
			if isCtx(node) {
				usage.usedOtherwise = true
			}
		}
		return true
	})

	return usage
}

// contextParamObject returns the variable of fn's named context parameter, or
// nil when the parameter is unnamed or blank.
func contextParamObject(pass *analysis.Pass, fn *ast.FuncDecl) *types.Var {
	for _, param := range fn.Type.Params.List {
		if !isContextTypeExpr(param.Type) || len(param.Names) == 0 {
			continue
		}
		if param.Names[0].Name == "_" {
			return nil
		}
		v, _ := pass.TypesInfo.Defs[param.Names[0]].(*types.Var)
		return v
	}
	return nil
}

// isContextTypeExpr reports whether a parameter's type expression is spelled
// as a context, the same test getContextParam uses.
func isContextTypeExpr(expr ast.Expr) bool {
	paramType := types.ExprString(expr)
	return strings.Contains(paramType, "context.Context") || paramType == "Context"
}

// trackContextCopies adds to ctxVars every variable on the left-hand side
// that is assigned a context variable as it is (c := ctx, var c = ctx).
func trackContextCopies(pass *analysis.Pass, lhs, rhs []ast.Expr, isCtx func(*ast.Ident) bool, ctxVars map[types.Object]bool) {
	if len(lhs) != len(rhs) {
		return
	}
	for i, value := range rhs {
		ident, ok := ast.Unparen(value).(*ast.Ident)
		if !ok || !isCtx(ident) {
			continue
		}
		target, ok := lhs[i].(*ast.Ident)
		if !ok {
			continue
		}
		if obj := pass.TypesInfo.ObjectOf(target); obj != nil {
			ctxVars[obj] = true
		}
	}
}

// storesIdentInField reports whether assign stores a context variable into a
// field selector (m.ctx, s.context, etc.).
func storesIdentInField(assign *ast.AssignStmt, isCtx func(*ast.Ident) bool) bool {
	for i, rhs := range assign.Rhs {
		ident, ok := rhs.(*ast.Ident)
		if !ok || !isCtx(ident) || i >= len(assign.Lhs) {
			continue
		}
		if _, ok := assign.Lhs[i].(*ast.SelectorExpr); ok {
			return true
		}
	}
	return false
}

// isContextMethodCall reports whether call is a context method call on a
// context variable (ctx.Done(), ctx.Err(), etc.).
func isContextMethodCall(call *ast.CallExpr, isCtx func(*ast.Ident) bool) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	return ok && isCtx(ident) && contextMethods[sel.Sel.Name]
}

// containsIdent checks if an expression contains an identifier matching isCtx
func containsIdent(expr ast.Expr, isCtx func(*ast.Ident) bool) bool {
	found := false
	ast.Inspect(expr, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok && isCtx(ident) {
			found = true
			return false
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
		if _, ok := n.(*ast.CallExpr); ok {
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
func checkCallsWithoutContext(pass *analysis.Pass, reporter *nolint.Reporter, fn *ast.FuncDecl) {
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
		if advice, isPkgCall := packageLevelCallsWithoutContext[callName]; isPkgCall {
			reporter.Reportf(call.Pos(),
				"%s called without context; %s", callName, advice)
		}

		// Check method calls that should use Context variants
		// Only flag if the first argument is NOT a context
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		methodName := sel.Sel.Name
		variant, ok := methodsRequiringContext[methodName]
		if ok && hasContextVariant(pass, sel, variant) && !firstArgIsContext(pass, call) {
			reporter.Reportf(call.Pos(),
				"%s() called without context as first argument; use %s instead", methodName, variant)
		}

		return true
	})
}

// hasContextVariant reports whether sel is a method call whose receiver also
// has the method variant, such as QueryContext next to Query.
func hasContextVariant(pass *analysis.Pass, sel *ast.SelectorExpr, variant string) bool {
	selection, ok := pass.TypesInfo.Selections[sel]
	if !ok || selection.Kind() != types.MethodVal {
		return false
	}
	obj, _, _ := types.LookupFieldOrMethod(selection.Recv(), true, pass.Pkg, variant)
	_, ok = obj.(*types.Func)
	return ok
}

// firstArgIsContext checks if the first argument to a call is a context: a
// value whose type has context.Context's methods, whatever it is called.
func firstArgIsContext(pass *analysis.Pass, call *ast.CallExpr) bool {
	if len(call.Args) == 0 {
		return false
	}
	t := pass.TypesInfo.TypeOf(call.Args[0])
	if t == nil {
		return false
	}
	for name := range contextMethods {
		obj, _, _ := types.LookupFieldOrMethod(t, true, pass.Pkg, name)
		if _, ok := obj.(*types.Func); !ok {
			return false
		}
	}
	return true
}

// checkContextAwareCalls checks for calls that have context-aware variants
func checkContextAwareCalls(reporter *nolint.Reporter, fn *ast.FuncDecl, hasContext bool) {
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

		// Check for time.Sleep when context is available. Methods with a
		// Context variant are checkCallsWithoutContext's job.
		if hasContext && callName == "time.Sleep" {
			reporter.Reportf(call.Pos(),
				"time.Sleep called when context is available; use select with <-ctx.Done() and time.After() instead")
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
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
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
