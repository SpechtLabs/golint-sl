// Package resourceclose provides an analyzer that detects resources that aren't properly closed.
// This includes HTTP response bodies, files, database connections, etc.
package resourceclose

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

// Doc is the analyzer's documentation.
const Doc = `detect resources that are not properly closed

This analyzer detects:
1. HTTP response bodies not closed (resp.Body.Close())
2. File handles not closed (file.Close())
3. Database rows not closed (rows.Close())
4. Network and gRPC connections not closed (conn.Close())

A resource counts as closed when its Close method is called anywhere in
the function: deferred, in a function literal, in a return statement or
as a plain call. A function that returns the resource, or a struct
literal holding it, hands it to its caller and is not reported.

Unclosed resources cause memory leaks, file descriptor exhaustion,
and connection pool starvation.`

// Analyzer reports opened resources that are never closed.
var Analyzer = &analysis.Analyzer{
	Name:     "resourceclose",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// resourcePattern defines a pattern for detecting unclosed resources
type resourcePattern struct {
	PkgPath     string   // package of the resource type, e.g. "net/http"
	TypeNames   []string // resource types in PkgPath, e.g. "Response"
	CloseField  string   // e.g., "Body" (empty means close on the var itself)
	Message     string   // Error message
	CreateFuncs []string // Functions that create this resource
}

// resourceKey identifies what has to be closed: a variable, or a field of
// it such as resp.Body.
type resourceKey struct {
	obj   types.Object
	field string
}

type resourceInfo struct {
	closeField string
	message    string
	pos        token.Pos
}

// closeMethod is the name of the method that releases a resource.
const closeMethod = "Close"

var patterns = []resourcePattern{
	{
		PkgPath:     "net/http",
		TypeNames:   []string{"Response"},
		CloseField:  "Body",
		Message:     "HTTP response body must be closed: defer resp.Body.Close()",
		CreateFuncs: []string{"Do", "Get", "Post", "Head", "PostForm", "RoundTrip"},
	},
	{
		PkgPath:     "os",
		TypeNames:   []string{"File"},
		Message:     "file must be closed: defer f.Close()",
		CreateFuncs: []string{"Open", "OpenFile", "Create", "CreateTemp"},
	},
	{
		PkgPath:     "database/sql",
		TypeNames:   []string{"Rows"},
		Message:     "database rows must be closed: defer rows.Close()",
		CreateFuncs: []string{"Query", "QueryRow", "QueryContext", "QueryRowContext"},
	},
	{
		PkgPath:     "database/sql",
		TypeNames:   []string{"Stmt"},
		Message:     "prepared statement must be closed: defer stmt.Close()",
		CreateFuncs: []string{"Prepare", "PrepareContext"},
	},
	{
		PkgPath:     "net",
		TypeNames:   []string{"Conn", "TCPConn", "UDPConn", "IPConn", "UnixConn"},
		Message:     "connection must be closed: defer conn.Close()",
		CreateFuncs: []string{"Dial", "DialContext", "DialTimeout", "DialTCP", "DialUDP", "DialIP", "DialUnix"},
	},
	{
		PkgPath:     "google.golang.org/grpc",
		TypeNames:   []string{"ClientConn"},
		Message:     "gRPC connection must be closed: defer conn.Close()",
		CreateFuncs: []string{"Dial", "DialContext", "NewClient"},
	},
}

func run(pass *analysis.Pass) (any, error) {
	reporter := nolint.NewReporter(pass)
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		fn := n.(*ast.FuncDecl)
		if fn.Body == nil {
			return
		}

		checkFunction(reporter, pass, fn)
	})

	return nil, nil
}

// checkFunction reports the resources fn opens and neither closes nor hands
// to its caller. Variables are tracked by their types.Object, so a shadowing
// variable of the same name is a resource of its own.
func checkFunction(reporter *nolint.Reporter, pass *analysis.Pass, fn *ast.FuncDecl) {
	resources := make(map[types.Object]resourceInfo)

	// released holds the resources that are closed or returned.
	released := make(map[resourceKey]bool)

	results := resultObjects(pass, fn)

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.AssignStmt:
			checkAssignment(pass, node, results, resources)
		case *ast.CallExpr:
			// A Close call in any position: deferred, plain, assigned,
			// returned, or inside a function literal such as t.Cleanup's.
			if key, ok := closeTarget(pass, node); ok {
				released[key] = true
			}
		case *ast.ReturnStmt:
			// Returning the resource hands it to the caller.
			for _, result := range node.Results {
				for _, key := range returnedResources(pass, result) {
					released[key] = true
				}
			}
		}
		return true
	})

	for obj, info := range resources {
		if released[resourceKey{obj: obj}] || released[resourceKey{obj: obj, field: info.closeField}] {
			continue
		}

		reporter.Reportf(info.pos, "%s", info.message)
	}
}

// resultObjects returns the named results of fn. A resource assigned to one
// of them is returned to the caller.
func resultObjects(pass *analysis.Pass, fn *ast.FuncDecl) map[types.Object]bool {
	objs := make(map[types.Object]bool)
	if fn.Type.Results == nil {
		return objs
	}

	for _, field := range fn.Type.Results.List {
		for _, name := range field.Names {
			if obj := pass.TypesInfo.Defs[name]; obj != nil {
				objs[obj] = true
			}
		}
	}

	return objs
}

func checkAssignment(pass *analysis.Pass, assign *ast.AssignStmt, results map[types.Object]bool, resources map[types.Object]resourceInfo) {
	for i, lhs := range assign.Lhs {
		ident, ok := lhs.(*ast.Ident)
		if !ok || ident.Name == "_" {
			continue
		}

		// Defs for :=, Uses for = to an existing variable.
		obj := pass.TypesInfo.ObjectOf(ident)
		if obj == nil || results[obj] {
			continue
		}

		// The call is the matching RHS, or the only RHS of a multi-value
		// call such as f, err := os.Open(name).
		rhs := assign.Rhs[0]
		if len(assign.Rhs) == len(assign.Lhs) {
			rhs = assign.Rhs[i]
		}

		// Check against patterns, using the variable's type rather than
		// the RHS expression's, which is a tuple for multi-value calls.
		if pattern, ok := matchResourcePattern(obj.Type(), callFuncName(rhs)); ok {
			resources[obj] = resourceInfo{
				pos:        assign.Pos(),
				closeField: pattern.CloseField,
				message:    pattern.Message,
			}
		}
	}
}

// matchResourcePattern returns the first resource pattern matching the variable type and create call
func matchResourcePattern(t types.Type, funcName string) (resourcePattern, bool) {
	if ptr, ok := types.Unalias(t).(*types.Pointer); ok {
		t = ptr.Elem()
	}

	named, ok := types.Unalias(t).(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return resourcePattern{}, false
	}

	pkgPath, typeName := named.Obj().Pkg().Path(), named.Obj().Name()

	for _, pattern := range patterns {
		if pattern.PkgPath != pkgPath || !slices.Contains(pattern.TypeNames, typeName) {
			continue
		}

		if !slices.Contains(pattern.CreateFuncs, funcName) {
			continue
		}

		return pattern, true
	}

	return resourcePattern{}, false
}

// callFuncName returns the name of the function expr calls, or "" when expr
// is not a call.
func callFuncName(expr ast.Expr) string {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return ""
	}

	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name
	case *ast.SelectorExpr:
		return fn.Sel.Name
	}

	return ""
}

// closeTarget returns the resource a Close call releases: x.Close() closes
// the variable x, x.Body.Close() the field Body of x.
func closeTarget(pass *analysis.Pass, call *ast.CallExpr) (resourceKey, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != closeMethod {
		return resourceKey{}, false
	}

	return resourceRef(pass, sel.X)
}

// returnedResources returns the resources a return statement's result hands
// to the caller: the resource itself (f, resp.Body), or the resources a
// struct literal holds (&Parser{f: f}).
func returnedResources(pass *analysis.Pass, result ast.Expr) []resourceKey {
	result = ast.Unparen(result)
	if unary, ok := result.(*ast.UnaryExpr); ok && unary.Op == token.AND {
		result = ast.Unparen(unary.X)
	}

	lit, ok := result.(*ast.CompositeLit)
	if !ok {
		if key, ok := resourceRef(pass, result); ok {
			return []resourceKey{key}
		}
		return nil
	}

	var keys []resourceKey
	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			elt = kv.Value
		}
		if key, ok := resourceRef(pass, elt); ok {
			keys = append(keys, key)
		}
	}

	return keys
}

// resourceRef resolves a variable x or a field selection x.Field to the
// variable's object.
func resourceRef(pass *analysis.Pass, expr ast.Expr) (resourceKey, bool) {
	switch e := ast.Unparen(expr).(type) {
	case *ast.Ident:
		if obj := pass.TypesInfo.Uses[e]; obj != nil {
			return resourceKey{obj: obj}, true
		}
	case *ast.SelectorExpr:
		if ident, ok := ast.Unparen(e.X).(*ast.Ident); ok {
			if obj := pass.TypesInfo.Uses[ident]; obj != nil {
				return resourceKey{obj: obj, field: e.Sel.Name}, true
			}
		}
	}

	return resourceKey{}, false
}
