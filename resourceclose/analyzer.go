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
	AssignType   string   // e.g., "*http.Response"
	CloseField   string   // e.g., "Body" (empty means close on the var itself)
	CloseCall    string   // e.g., "Close"
	Message      string   // Error message
	CreateFuncs  []string // Functions that create this resource (if empty, match by type only)
}

var patterns = []resourcePattern{
	{
		AssignType:  "http.Response",
		CloseField:  "Body",
		CloseCall:   "Close",
		Message:     "HTTP response body must be closed: defer resp.Body.Close()",
		CreateFuncs: []string{"Do", "Get", "Post", "Head", "PostForm", "RoundTrip"},
	},
	{
		AssignType:  "os.File",
		CloseField:  "",
		CloseCall:   "Close",
		Message:     "file must be closed: defer f.Close()",
		CreateFuncs: []string{"Open", "OpenFile", "Create", "CreateTemp"},
	},
	{
		AssignType:  "sql.Rows",
		CloseField:  "",
		CloseCall:   "Close",
		Message:     "database rows must be closed: defer rows.Close()",
		CreateFuncs: []string{"Query", "QueryRow", "QueryContext", "QueryRowContext"},
	},
	{
		AssignType:  "sql.Stmt",
		CloseField:  "",
		CloseCall:   "Close",
		Message:     "prepared statement must be closed: defer stmt.Close()",
		CreateFuncs: []string{"Prepare", "PrepareContext"},
	},
	{
		AssignType:  "net.Conn",
		CloseField:  "",
		CloseCall:   "Close",
		Message:     "connection must be closed: defer conn.Close()",
		CreateFuncs: []string{"Dial", "DialContext", "DialTimeout", "DialTCP", "DialUDP", "DialIP", "DialUnix"},
	},
	{
		AssignType:  "grpc.ClientConn",
		CloseField:  "",
		CloseCall:   "Close",
		Message:     "gRPC connection must be closed: defer conn.Close()",
		CreateFuncs: []string{"Dial", "DialContext", "NewClient"},
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

	// First pass: find all resource assignments and closes
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.AssignStmt:
			checkAssignment(pass, node, resourceVars)
			// Also check for close calls on RHS: _ = f.Close()
			for _, rhs := range node.Rhs {
				if call, ok := rhs.(*ast.CallExpr); ok {
					checkCloseCall(call, closedResources)
				}
			}
		case *ast.DeferStmt:
			checkDefer(node, closedResources)
		case *ast.ExprStmt:
			// Non-deferred close calls
			if call, ok := node.X.(*ast.CallExpr); ok {
				checkCloseCall(call, closedResources)
				// Check for t.Cleanup(func() { ... }) patterns
				checkTestCleanup(call, closedResources)
			}
		case *ast.IfStmt:
			// Check for closes inside if blocks (common pattern for create-and-close)
			checkIfBlockCloses(node, closedResources)
			// Check for pattern: if f, err := os.Create(...); err == nil { f.Close() }
			checkIfInitAndClose(node, closedResources)
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

// checkTestCleanup checks for t.Cleanup(func() { ... Close() ... }) patterns
func checkTestCleanup(call *ast.CallExpr, closedResources map[string]bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	// Check for t.Cleanup or similar cleanup patterns
	if sel.Sel.Name != "Cleanup" {
		return
	}

	// Check the argument (should be a function literal)
	if len(call.Args) != 1 {
		return
	}

	funcLit, ok := call.Args[0].(*ast.FuncLit)
	if !ok {
		return
	}

	// Look for Close calls inside the cleanup function
	ast.Inspect(funcLit.Body, func(n ast.Node) bool {
		if callExpr, ok := n.(*ast.CallExpr); ok {
			checkCloseCall(callExpr, closedResources)
		}
		return true
	})
}

// checkIfBlockCloses checks for closes inside if blocks
func checkIfBlockCloses(ifStmt *ast.IfStmt, closedResources map[string]bool) {
	// Check the if body
	ast.Inspect(ifStmt.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.ExprStmt:
			if call, ok := node.X.(*ast.CallExpr); ok {
				checkCloseCall(call, closedResources)
			}
		case *ast.AssignStmt:
			// Handle: _ = f.Close()
			for _, rhs := range node.Rhs {
				if call, ok := rhs.(*ast.CallExpr); ok {
					checkCloseCall(call, closedResources)
				}
			}
		case *ast.DeferStmt:
			checkDefer(node, closedResources)
		}
		return true
	})

	// Check the else block if present
	if ifStmt.Else != nil {
		ast.Inspect(ifStmt.Else, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.ExprStmt:
				if call, ok := node.X.(*ast.CallExpr); ok {
					checkCloseCall(call, closedResources)
				}
			case *ast.AssignStmt:
				// Handle: _ = f.Close()
				for _, rhs := range node.Rhs {
					if call, ok := rhs.(*ast.CallExpr); ok {
						checkCloseCall(call, closedResources)
					}
				}
			case *ast.DeferStmt:
				checkDefer(node, closedResources)
			}
			return true
		})
	}
}

// checkIfInitAndClose handles pattern: if f, err := os.Create(...); err == nil { f.Close() }
// where the resource is created in the if's init clause and closed in the body
func checkIfInitAndClose(ifStmt *ast.IfStmt, closedResources map[string]bool) {
	// Check if the if statement has an init clause with an assignment
	if ifStmt.Init == nil {
		return
	}

	assign, ok := ifStmt.Init.(*ast.AssignStmt)
	if !ok {
		return
	}

	// Get the variable names from the assignment
	varNames := make(map[string]bool)
	for _, lhs := range assign.Lhs {
		if ident, ok := lhs.(*ast.Ident); ok && ident.Name != "_" {
			varNames[ident.Name] = true
		}
	}

	// Look for closes of these variables in the if body
	ast.Inspect(ifStmt.Body, func(n ast.Node) bool {
		var call *ast.CallExpr

		switch node := n.(type) {
		case *ast.ExprStmt:
			call, _ = node.X.(*ast.CallExpr)
		case *ast.AssignStmt:
			// Handle: _ = f.Close()
			if len(node.Rhs) == 1 {
				call, _ = node.Rhs[0].(*ast.CallExpr)
			}
		}

		if call != nil {
			if target := getCloseTarget(call); target != "" {
				// Check if this closes one of the init variables
				parts := strings.Split(target, ".")
				if len(parts) > 0 && varNames[parts[0]] {
					closedResources[target] = true
					closedResources[parts[0]] = true
				}
			}
		}
		return true
	})
}

// getCloseTarget returns the target of a Close() call, or empty string if not a Close call
func getCloseTarget(call *ast.CallExpr) string {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}

	if sel.Sel.Name != "Close" {
		return ""
	}

	return exprToString(sel.X)
}

type resourceInfo struct {
	pos        token.Pos
	closeField string
	message    string
}

// isStdioAssignment checks if the RHS is os.Stdout, os.Stderr, or os.Stdin
func isStdioAssignment(rhs ast.Expr) bool {
	sel, ok := rhs.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	if ident.Name != "os" {
		return false
	}
	switch sel.Sel.Name {
	case "Stdout", "Stderr", "Stdin":
		return true
	}
	return false
}

func checkAssignment(pass *analysis.Pass, assign *ast.AssignStmt, resourceVars map[string]resourceInfo) {
	// Check each assigned variable
	for i, lhs := range assign.Lhs {
		ident, ok := lhs.(*ast.Ident)
		if !ok || ident.Name == "_" {
			continue
		}

		// Get the RHS expression
		var rhsExpr ast.Expr
		if i < len(assign.Rhs) {
			rhsExpr = assign.Rhs[i]
		} else if len(assign.Rhs) == 1 {
			rhsExpr = assign.Rhs[0]
		}

		// Skip os.Stdout, os.Stderr, os.Stdin - these shouldn't be closed
		if rhsExpr != nil && isStdioAssignment(rhsExpr) {
			continue
		}

		// Get the type of the RHS if possible
		var rhsType types.Type
		if rhsExpr != nil {
			rhsType = pass.TypesInfo.TypeOf(rhsExpr)
		}

		if rhsType == nil {
			continue
		}

		// Get the function being called (if any)
		callFuncName := getCallFuncName(assign.Rhs)

		// Check against patterns
		typeStr := rhsType.String()
		for _, pattern := range patterns {
			if strings.Contains(typeStr, pattern.AssignType) {
				// If pattern has CreateFuncs, only match if the call matches
				if len(pattern.CreateFuncs) > 0 {
					if !isCreateFunc(callFuncName, pattern.CreateFuncs) {
						continue
					}
				}
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

// getCallFuncName extracts the function name from a call expression in the RHS
func getCallFuncName(rhs []ast.Expr) string {
	if len(rhs) == 0 {
		return ""
	}

	// Handle the first RHS (for multiple returns, the call is always on the first)
	call, ok := rhs[0].(*ast.CallExpr)
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

// isCreateFunc checks if the function name matches any of the create functions
func isCreateFunc(funcName string, createFuncs []string) bool {
	for _, cf := range createFuncs {
		if funcName == cf {
			return true
		}
	}
	return false
}

func checkDefer(deferStmt *ast.DeferStmt, closedResources map[string]bool) {
	// Handle direct defer: defer resp.Body.Close()
	checkCloseCall(deferStmt.Call, closedResources)

	// Handle defer with function literal: defer func() { resp.Body.Close() }()
	if call, ok := deferStmt.Call.Fun.(*ast.FuncLit); ok {
		ast.Inspect(call.Body, func(n ast.Node) bool {
			if callExpr, ok := n.(*ast.CallExpr); ok {
				checkCloseCall(callExpr, closedResources)
			}
			return true
		})
	}
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
