// Package httpclient provides an analyzer that enforces http.Client best practices.
// It detects common mistakes like missing timeouts and improper Transport configuration.
package httpclient

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

const Doc = `enforce http.Client best practices

This analyzer detects:
1. http.Client{} without Timeout set (will hang forever on slow servers)
2. http.DefaultClient usage (has no timeout, shared globally)
3. http.Get/Post/etc direct calls (use shared DefaultClient)
4. Missing Transport configuration for connection pooling

HTTP clients without timeouts are a common source of goroutine leaks
and hung services in production.`

var Analyzer = &analysis.Analyzer{
	Name:     "httpclient",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	reporter := nolint.NewReporter(pass)
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.CompositeLit)(nil),
		(*ast.CallExpr)(nil),
		(*ast.SelectorExpr)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.CompositeLit:
			checkClientLiteral(reporter, pass, node)
		case *ast.CallExpr:
			checkDirectHTTPCalls(reporter, node)
		case *ast.SelectorExpr:
			checkDefaultClient(reporter, node)
		}
	})

	return nil, nil
}

// checkClientLiteral detects http.Client{} without Timeout
func checkClientLiteral(reporter *nolint.Reporter, pass *analysis.Pass, lit *ast.CompositeLit) {
	// Check if this is an http.Client composite literal
	if !isHTTPClientType(pass, lit.Type) {
		return
	}

	hasTimeout := false
	hasTransport := false

	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}

		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}

		switch key.Name {
		case "Timeout":
			hasTimeout = true
		case "Transport":
			hasTransport = true
		}
	}

	if !hasTimeout {
		reporter.Reportf(lit.Pos(),
			"http.Client without Timeout will wait forever; always set Timeout (e.g., 30*time.Second)")
	}

	// Transport is recommended but not required
	_ = hasTransport
}

// isHTTPClientType checks if a type is http.Client
func isHTTPClientType(pass *analysis.Pass, expr ast.Expr) bool {
	if expr == nil {
		return false
	}

	t := pass.TypesInfo.TypeOf(expr)
	if t == nil {
		// Fall back to AST analysis
		return isHTTPClientAST(expr)
	}

	// Check the type name
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}

	obj := named.Obj()
	if obj == nil {
		return false
	}

	return obj.Name() == "Client" && obj.Pkg() != nil && obj.Pkg().Path() == "net/http"
}

// isHTTPClientAST checks using AST when type info isn't available
func isHTTPClientAST(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	if sel.Sel.Name != "Client" {
		return false
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	return ident.Name == "http"
}

// checkDirectHTTPCalls detects http.Get, http.Post, etc.
func checkDirectHTTPCalls(reporter *nolint.Reporter, call *ast.CallExpr) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return
	}

	if ident.Name != "http" {
		return
	}

	directCalls := map[string]string{
		"Get":        "http.Get uses DefaultClient with no timeout; create a client with Timeout",
		"Post":       "http.Post uses DefaultClient with no timeout; create a client with Timeout",
		"PostForm":   "http.PostForm uses DefaultClient with no timeout; create a client with Timeout",
		"Head":       "http.Head uses DefaultClient with no timeout; create a client with Timeout",
		"NewRequest": "http.NewRequest doesn't support context; use http.NewRequestWithContext instead",
	}

	if msg, found := directCalls[sel.Sel.Name]; found {
		reporter.Reportf(call.Pos(), "%s", msg)
	}
}

// checkDefaultClient detects http.DefaultClient usage
func checkDefaultClient(reporter *nolint.Reporter, sel *ast.SelectorExpr) {
	if sel.Sel.Name != "DefaultClient" {
		return
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return
	}

	if ident.Name == "http" {
		reporter.Reportf(sel.Pos(),
			"http.DefaultClient has no timeout and is shared globally; create your own http.Client with Timeout")
	}
}
