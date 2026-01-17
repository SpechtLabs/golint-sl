// Package closurecomplexity provides an analyzer that detects overly complex closures.
//
// Anonymous functions (closures) should be kept simple. Complex business logic
// should be extracted into named functions for better readability and testability.
package closurecomplexity

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

const Doc = `detect overly complex anonymous functions (closures)

Closures should be kept simple. Complex business logic should be
extracted into named functions for:
1. Better readability
2. Easier testing
3. Reusability
4. Better stack traces in errors

Good pattern:
    // Simple closure for goroutine
    go func() {
        result <- processItem(item)
    }()

    // Named function for complex logic
    func processItem(item Item) Result {
        // complex logic here
    }

Bad pattern:
    go func() {
        // 30 lines of complex business logic
        // nested ifs, loops, error handling
        // impossible to test in isolation
    }()

This analyzer flags:
1. Closures with more than 10 statements
2. Closures with nesting depth > 2
3. Closures capturing many variables (> 3)

Note: Test files are skipped, as table-driven tests commonly use
longer closures for setup, fixtures, and mock configuration.`

var Analyzer = &analysis.Analyzer{
	Name:     "closurecomplexity",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

const (
	// MaxClosureStatements is the maximum statements allowed in a closure
	MaxClosureStatements = 15
	// MaxClosureNesting is the maximum nesting depth in a closure
	MaxClosureNesting = 2
	// MaxCapturedVars is the maximum variables captured from outer scope
	MaxCapturedVars = 5
)

// exemptCobraFields are struct fields in Cobra commands that commonly have large closures
var exemptCobraFields = map[string]bool{
	"RunE":              true,
	"Run":               true,
	"PreRunE":           true,
	"PreRun":            true,
	"PostRunE":          true,
	"PostRun":           true,
	"PersistentPreRunE": true,
	"PersistentPreRun":  true,
}

// exemptHTTPFields are struct fields for HTTP handlers
var exemptHTTPFields = map[string]bool{
	"Handler":     true,
	"HandlerFunc": true,
}

// exemptVisitorFuncs are function names that take visitor/callback closures
// These callbacks naturally need to handle all the logic for each visited node
var exemptVisitorFuncs = map[string]bool{
	// AST visitor patterns
	"Inspect":  true,
	"Preorder": true,
	"Walk":     true,
	// Flag visitor patterns
	"VisitAll": true,
	"Visit":    true,
	// Filepath walking
	"WalkDir":  true,
	"WalkFunc": true,
	// Generic iteration patterns
	"ForEach": true,
	"Range":   true,
}

func run(pass *analysis.Pass) (interface{}, error) {
	reporter := nolint.NewReporter(pass)
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	var currentFunc *ast.FuncDecl
	var inTestFile bool

	// Track closures that should be exempt
	exemptClosures := make(map[*ast.FuncLit]bool)

	// First pass: find exempt closures
	nodeFilter := []ast.Node{
		(*ast.File)(nil),
		(*ast.FuncDecl)(nil),
		(*ast.DeferStmt)(nil),
		(*ast.KeyValueExpr)(nil),
		(*ast.ReturnStmt)(nil),
		(*ast.GoStmt)(nil),
		(*ast.CallExpr)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.File:
			filename := pass.Fset.Position(node.Pos()).Filename
			inTestFile = strings.HasSuffix(filename, "_test.go")

		case *ast.FuncDecl:
			currentFunc = node

		case *ast.DeferStmt:
			// Exempt deferred closures - they're commonly used for cleanup/telemetry
			if funcLit, ok := node.Call.Fun.(*ast.FuncLit); ok {
				exemptClosures[funcLit] = true
			}

		case *ast.GoStmt:
			// Exempt goroutine closures - they need to capture context
			if funcLit, ok := node.Call.Fun.(*ast.FuncLit); ok {
				exemptClosures[funcLit] = true
			}

		case *ast.ReturnStmt:
			// Exempt closures returned from functions (handler factory pattern)
			for _, result := range node.Results {
				if funcLit, ok := result.(*ast.FuncLit); ok {
					exemptClosures[funcLit] = true
				}
			}

		case *ast.CallExpr:
			// Check for visitor pattern callbacks (e.g., ast.Inspect, f.VisitAll)
			funcName := getCallFuncName(node)
			if exemptVisitorFuncs[funcName] {
				for _, arg := range node.Args {
					if funcLit, ok := arg.(*ast.FuncLit); ok {
						exemptClosures[funcLit] = true
					}
				}
			}

		case *ast.KeyValueExpr:
			// Check for Cobra RunE/Run and HTTP handler fields
			if ident, ok := node.Key.(*ast.Ident); ok {
				if exemptCobraFields[ident.Name] || exemptHTTPFields[ident.Name] {
					if funcLit, ok := node.Value.(*ast.FuncLit); ok {
						exemptClosures[funcLit] = true
					}
				}
			}
		}
	})

	// Second pass: check non-exempt closures
	closureFilter := []ast.Node{
		(*ast.File)(nil),
		(*ast.FuncDecl)(nil),
		(*ast.FuncLit)(nil),
	}

	inspect.Preorder(closureFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.File:
			filename := pass.Fset.Position(node.Pos()).Filename
			inTestFile = strings.HasSuffix(filename, "_test.go")

		case *ast.FuncDecl:
			currentFunc = node

		case *ast.FuncLit:
			if inTestFile {
				return // Skip closures in test files
			}
			if exemptClosures[node] {
				return // Skip exempt closures
			}
			checkClosure(reporter, node, currentFunc)
		}
	})

	return nil, nil
}

func checkClosure(reporter *nolint.Reporter, closure *ast.FuncLit, parentFunc *ast.FuncDecl) {
	if closure.Body == nil {
		return
	}

	// Count statements
	stmtCount := countStatements(closure.Body)
	if stmtCount > MaxClosureStatements {
		reporter.Reportf(closure.Pos(),
			"closure has %d statements (max %d); extract complex logic into a named function for testability",
			stmtCount, MaxClosureStatements)
	}

	// Check nesting depth
	depth := maxNestingDepth(closure.Body, 0)
	if depth > MaxClosureNesting {
		reporter.Reportf(closure.Pos(),
			"closure has nesting depth of %d (max %d); extract into a named function",
			depth, MaxClosureNesting)
	}

	// Count captured variables
	if parentFunc != nil {
		captured := countCapturedVars(closure, parentFunc)
		if captured > MaxCapturedVars {
			reporter.Reportf(closure.Pos(),
				"closure captures %d variables from outer scope (max %d); consider passing them as parameters or extracting to a named function",
				captured, MaxCapturedVars)
		}
	}
}

func countStatements(block *ast.BlockStmt) int {
	count := 0
	ast.Inspect(block, func(n ast.Node) bool {
		switch n.(type) {
		case ast.Stmt:
			count++
		case *ast.FuncLit:
			// Don't count nested closures
			return false
		}
		return true
	})
	return count
}

func maxNestingDepth(node ast.Node, current int) int {
	maxDepth := current

	// Get the body to inspect based on node type
	var body *ast.BlockStmt
	switch n := node.(type) {
	case *ast.BlockStmt:
		body = n
	case *ast.IfStmt:
		body = n.Body
	case *ast.ForStmt:
		body = n.Body
	case *ast.RangeStmt:
		body = n.Body
	case *ast.SwitchStmt:
		body = n.Body
	case *ast.TypeSwitchStmt:
		body = n.Body
	case *ast.SelectStmt:
		body = n.Body
	default:
		return current
	}

	if body == nil {
		return current
	}

	for _, stmt := range body.List {
		switch s := stmt.(type) {
		case *ast.IfStmt:
			depth := maxNestingDepth(s, current+1)
			if depth > maxDepth {
				maxDepth = depth
			}
			// Check else branch
			if s.Else != nil {
				if elseIf, ok := s.Else.(*ast.IfStmt); ok {
					depth = maxNestingDepth(elseIf, current+1)
				} else if elseBlock, ok := s.Else.(*ast.BlockStmt); ok {
					depth = maxNestingDepth(elseBlock, current)
				}
				if depth > maxDepth {
					maxDepth = depth
				}
			}

		case *ast.ForStmt:
			depth := maxNestingDepth(s, current+1)
			if depth > maxDepth {
				maxDepth = depth
			}

		case *ast.RangeStmt:
			depth := maxNestingDepth(s, current+1)
			if depth > maxDepth {
				maxDepth = depth
			}

		case *ast.SwitchStmt:
			depth := maxNestingDepth(s, current+1)
			if depth > maxDepth {
				maxDepth = depth
			}

		case *ast.TypeSwitchStmt:
			depth := maxNestingDepth(s, current+1)
			if depth > maxDepth {
				maxDepth = depth
			}

		case *ast.SelectStmt:
			depth := maxNestingDepth(s, current+1)
			if depth > maxDepth {
				maxDepth = depth
			}

		case *ast.BlockStmt:
			depth := maxNestingDepth(s, current)
			if depth > maxDepth {
				maxDepth = depth
			}
		}
	}

	return maxDepth
}

func countCapturedVars(closure *ast.FuncLit, parentFunc *ast.FuncDecl) int {
	// Get closure parameters (not captured)
	params := make(map[string]bool)
	if closure.Type.Params != nil {
		for _, field := range closure.Type.Params.List {
			for _, name := range field.Names {
				params[name.Name] = true
			}
		}
	}

	// Get parent function's local variables
	parentVars := collectLocalVars(parentFunc)

	// Find variables used in closure that come from parent
	captured := make(map[string]bool)
	ast.Inspect(closure.Body, func(n ast.Node) bool {
		ident, ok := n.(*ast.Ident)
		if !ok {
			return true
		}

		// Skip if it's a parameter
		if params[ident.Name] {
			return true
		}

		// Skip common non-captured identifiers
		if isBuiltinOrCommon(ident.Name) {
			return true
		}

		// Check if it's from parent scope
		if parentVars[ident.Name] {
			captured[ident.Name] = true
		}

		return true
	})

	return len(captured)
}

func collectLocalVars(fn *ast.FuncDecl) map[string]bool {
	vars := make(map[string]bool)

	if fn == nil || fn.Body == nil {
		return vars
	}

	// Add parameters
	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			for _, name := range field.Names {
				vars[name.Name] = true
			}
		}
	}

	// Add local variables
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.AssignStmt:
			for _, lhs := range node.Lhs {
				if ident, ok := lhs.(*ast.Ident); ok {
					vars[ident.Name] = true
				}
			}
		case *ast.ValueSpec:
			for _, name := range node.Names {
				vars[name.Name] = true
			}
		case *ast.FuncLit:
			// Don't recurse into closures
			return false
		}
		return true
	})

	return vars
}

func isBuiltinOrCommon(name string) bool {
	builtins := map[string]bool{
		// Builtins
		"nil": true, "true": true, "false": true,
		"append": true, "cap": true, "close": true, "complex": true,
		"copy": true, "delete": true, "imag": true, "len": true,
		"make": true, "new": true, "panic": true, "print": true,
		"println": true, "real": true, "recover": true,
		// Common types
		"error": true, "string": true, "int": true, "bool": true,
		"byte": true, "rune": true, "float64": true, "float32": true,
		// Common packages (when used as selectors)
		"fmt": true, "log": true, "time": true, "context": true,
		"http": true, "json": true, "errors": true, "strings": true,
	}
	return builtins[name]
}

// getCallFuncName extracts the function name from a call expression
func getCallFuncName(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name
	case *ast.SelectorExpr:
		return fn.Sel.Name
	}
	return ""
}
