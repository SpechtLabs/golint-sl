// Package httpclient provides an analyzer that enforces http.Client best practices.
// It detects common mistakes like missing timeouts and requests without a context.
package httpclient

import (
	"go/ast"
	"go/constant"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

// httpPkgPath is the import path of net/http.
const httpPkgPath = "net/http"

// Doc is the httpclient analyzer's documentation.
const Doc = `enforce http.Client best practices

This analyzer detects:
1. http.Client{} without Timeout set, or with a constant zero Timeout
   (will hang forever on slow servers)
2. http.DefaultClient usage (has no timeout, shared globally)
3. http.Get/Post/PostForm/Head direct calls (use shared DefaultClient)
4. http.NewRequest calls (no context; use http.NewRequestWithContext)

net/http is recognized through the type checker, so a renamed import or a
type alias of http.Client is checked too.

HTTP clients without timeouts are a common source of goroutine leaks
and hung services in production.`

// Analyzer enforces http.Client best practices.
var Analyzer = &analysis.Analyzer{
	Name:     "httpclient",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// directCalls maps the net/http functions that are reported to their message.
var directCalls = map[string]string{
	"Get":        "http.Get uses DefaultClient with no timeout; create a client with Timeout",
	"Post":       "http.Post uses DefaultClient with no timeout; create a client with Timeout",
	"PostForm":   "http.PostForm uses DefaultClient with no timeout; create a client with Timeout",
	"Head":       "http.Head uses DefaultClient with no timeout; create a client with Timeout",
	"NewRequest": "http.NewRequest doesn't support context; use http.NewRequestWithContext instead",
}

func run(pass *analysis.Pass) (any, error) {
	reporter := nolint.NewReporter(pass)
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.CompositeLit)(nil),
		(*ast.CallExpr)(nil),
		(*ast.SelectorExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.CompositeLit:
			checkClientLiteral(reporter, pass, node)
		case *ast.CallExpr:
			checkDirectHTTPCalls(reporter, pass, node)
		case *ast.SelectorExpr:
			checkDefaultClient(reporter, pass, node)
		}
	})

	return nil, nil
}

// checkClientLiteral detects http.Client{} without Timeout
func checkClientLiteral(reporter *nolint.Reporter, pass *analysis.Pass, lit *ast.CompositeLit) {
	// Check if this is an http.Client composite literal. An element literal
	// with an elided type, as in []*http.Client{{...}}, has the pointer type.
	t := pass.TypesInfo.TypeOf(lit)
	if ptr, ok := types.Unalias(t).(*types.Pointer); ok && lit.Type == nil {
		t = ptr.Elem()
	}
	client, ok := httpClientStruct(t)
	if !ok {
		return
	}

	if !setsTimeout(pass, client, lit) {
		reporter.Reportf(lit.Pos(),
			"http.Client without Timeout will wait forever; always set Timeout (e.g., 30*time.Second)")
	}
}

// httpClientStruct returns the struct type of t if t is net/http's Client,
// directly or through an alias.
func httpClientStruct(t types.Type) (*types.Struct, bool) {
	named, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return nil, false
	}

	obj := named.Obj()
	if obj.Name() != "Client" || obj.Pkg() == nil || obj.Pkg().Path() != httpPkgPath {
		return nil, false
	}

	st, ok := named.Underlying().(*types.Struct)
	return st, ok
}

// setsTimeout reports whether the http.Client literal sets Timeout to
// something other than a constant zero, by key or by position.
func setsTimeout(pass *analysis.Pass, client *types.Struct, lit *ast.CompositeLit) bool {
	for i, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "Timeout" {
				return !isConstantZero(pass, kv.Value)
			}
			continue
		}

		// An unkeyed literal lists every field in declaration order.
		if i < client.NumFields() && client.Field(i).Name() == "Timeout" {
			return !isConstantZero(pass, elt)
		}
	}
	return false
}

// isConstantZero reports whether expr is a constant equal to zero.
func isConstantZero(pass *analysis.Pass, expr ast.Expr) bool {
	tv, ok := pass.TypesInfo.Types[expr]
	return ok && tv.Value != nil && constant.Sign(tv.Value) == 0
}

// checkDirectHTTPCalls detects http.Get, http.Post, etc.
func checkDirectHTTPCalls(reporter *nolint.Reporter, pass *analysis.Pass, call *ast.CallExpr) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	fn, ok := pass.TypesInfo.Uses[sel.Sel].(*types.Func)
	if !ok || !isHTTPPackageLevel(fn) {
		return
	}

	if msg, found := directCalls[fn.Name()]; found {
		reporter.Reportf(call.Pos(), "%s", msg)
	}
}

// checkDefaultClient detects http.DefaultClient usage
func checkDefaultClient(reporter *nolint.Reporter, pass *analysis.Pass, sel *ast.SelectorExpr) {
	if sel.Sel.Name != "DefaultClient" {
		return
	}

	v, ok := pass.TypesInfo.Uses[sel.Sel].(*types.Var)
	if ok && isHTTPPackageLevel(v) {
		reporter.Reportf(sel.Pos(),
			"http.DefaultClient has no timeout and is shared globally; create your own http.Client with Timeout")
	}
}

// isHTTPPackageLevel reports whether obj is declared at the package level of
// net/http, which excludes methods and struct fields of the same name.
func isHTTPPackageLevel(obj types.Object) bool {
	pkg := obj.Pkg()
	return pkg != nil && pkg.Path() == httpPkgPath && obj.Parent() == pkg.Scope()
}
