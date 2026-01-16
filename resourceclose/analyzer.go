// Package resourceclose provides an analyzer that detects resources that aren't properly closed.
// This includes HTTP response bodies, files, database connections, etc.
package resourceclose

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

const Doc = `detect resources that are not properly closed

This analyzer detects:
1. HTTP response bodies not closed (resp.Body.Close())
2. File handles not closed (file.Close())
3. Database rows not closed (rows.Close())
4. gRPC streams not closed

Unclosed resources cause memory leaks, file descriptor exhaustion,
and connection pool starvation.`

var Analyzer = &analysis.Analyzer{
	Name:     "resourceclose",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// resourcePattern defines a pattern for detecting unclosed resources
type resourcePattern struct {
	AssignType string // e.g., "*http.Response"
	CloseField string // e.g., "Body" (empty means close on the var itself)
	CloseCall  string // e.g., "Close"
	Message    string // Error message
}

var patterns = []resourcePattern{
	{
		AssignType: "Response",
		CloseField: "Body",
		CloseCall:  "Close",
		Message:    "HTTP response body must be closed: defer resp.Body.Close()",
	},
	{
		AssignType: "File",
		CloseField: "",
		CloseCall:  "Close",
		Message:    "file must be closed: defer f.Close()",
	},
	{
		AssignType: "Rows",
		CloseField: "",
		CloseCall:  "Close",
		Message:    "database rows must be closed: defer rows.Close()",
	},
	{
		AssignType: "Stmt",
		CloseField: "",
		CloseCall:  "Close",
		Message:    "prepared statement must be closed: defer stmt.Close()",
	},
	{
		AssignType: "Conn",
		CloseField: "",
		CloseCall:  "Close",
		Message:    "connection must be closed: defer conn.Close()",
	},
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

		checkFunction(reporter, pass, fn)
	})

	return nil, nil
}

func checkFunction(reporter *nolint.Reporter, pass *analysis.Pass, fn *ast.FuncDecl) {
	// Track variables that hold closeable resources
	resourceVars := make(map[string]resourceInfo)

	// Track which resources have been closed
	closedResources := make(map[string]bool)

	// First pass: find all resource assignments
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.AssignStmt:
			checkAssignment(pass, node, resourceVars)
		case *ast.DeferStmt:
			checkDefer(node, closedResources)
		case *ast.ExprStmt:
			// Non-deferred close calls
			if call, ok := node.X.(*ast.CallExpr); ok {
				checkCloseCall(call, closedResources)
			}
		}
		return true
	})

	// Report unclosed resources
	for varName, info := range resourceVars {
		closeKey := varName
		if info.closeField != "" {
			closeKey = varName + "." + info.closeField
		}

		if !closedResources[closeKey] && !closedResources[varName] {
			reporter.Reportf(info.pos, "%s", info.message)
		}
	}
}

type resourceInfo struct {
	pos        token.Pos
	closeField string
	message    string
}

func checkAssignment(pass *analysis.Pass, assign *ast.AssignStmt, resourceVars map[string]resourceInfo) {
	// Check each assigned variable
	for i, lhs := range assign.Lhs {
		ident, ok := lhs.(*ast.Ident)
		if !ok || ident.Name == "_" {
			continue
		}

		// Get the type of the RHS if possible
		var rhsType types.Type
		if i < len(assign.Rhs) {
			rhsType = pass.TypesInfo.TypeOf(assign.Rhs[i])
		} else if len(assign.Rhs) == 1 {
			// Multiple return values
			if tuple, ok := pass.TypesInfo.TypeOf(assign.Rhs[0]).(*types.Tuple); ok {
				if i < tuple.Len() {
					rhsType = tuple.At(i).Type()
				}
			}
		}

		if rhsType == nil {
			continue
		}

		// Check against patterns
		typeStr := rhsType.String()
		for _, pattern := range patterns {
			if strings.Contains(typeStr, pattern.AssignType) {
				resourceVars[ident.Name] = resourceInfo{
					pos:        assign.Pos(),
					closeField: pattern.CloseField,
					message:    pattern.Message,
				}
				break
			}
		}
	}
}

func checkDefer(deferStmt *ast.DeferStmt, closedResources map[string]bool) {
	checkCloseCall(deferStmt.Call, closedResources)
}

func checkCloseCall(call *ast.CallExpr, closedResources map[string]bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	if sel.Sel.Name != "Close" {
		return
	}

	// Get what's being closed
	closeTarget := exprToString(sel.X)
	if closeTarget != "" {
		closedResources[closeTarget] = true
	}
}

func exprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		base := exprToString(e.X)
		if base != "" {
			return base + "." + e.Sel.Name
		}
		return e.Sel.Name
	}
	return ""
}
