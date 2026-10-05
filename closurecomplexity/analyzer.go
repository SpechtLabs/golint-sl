// Package closurecomplexity provides an analyzer that detects overly complex closures.
//
// Anonymous functions (closures) should be kept simple. Complex business logic
// should be extracted into named functions for better readability and testability.
package closurecomplexity

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

// Doc is the closurecomplexity analyzer's documentation.
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
1. Closures with more than 15 statements
2. Closures with nesting depth > 2
3. Closures capturing more than 5 variables from an enclosing function

Note: Test files are skipped, as table-driven tests commonly use
longer closures for setup, fixtures, and mock configuration.`

// Analyzer reports overly complex anonymous functions.
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

func run(pass *analysis.Pass) (any, error) {
	reporter := nolint.NewReporter(pass)
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	var inTestFile bool

	// Track closures that should be exempt
	exemptClosures := make(map[*ast.FuncLit]bool)

	// First pass: find exempt closures
	nodeFilter := []ast.Node{
		(*ast.DeferStmt)(nil),
		(*ast.KeyValueExpr)(nil),
		(*ast.ReturnStmt)(nil),
		(*ast.GoStmt)(nil),
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		markExemptClosures(n, exemptClosures)
	})

	// Second pass: check non-exempt closures
	closureFilter := []ast.Node{
		(*ast.File)(nil),
		(*ast.FuncLit)(nil),
	}

	insp.Preorder(closureFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.File:
			filename := pass.Fset.Position(node.Pos()).Filename
			inTestFile = strings.HasSuffix(filename, "_test.go")

		case *ast.FuncLit:
			if inTestFile {
				return // Skip closures in test files
			}
			if exemptClosures[node] {
				return // Skip exempt closures
			}
			checkClosure(pass, reporter, node)
		}
	})

	return nil, nil
}

// markExemptClosures records the closures under n that are exempt from the
// complexity checks because of where they appear.
func markExemptClosures(n ast.Node, exempt map[*ast.FuncLit]bool) {
	switch node := n.(type) {
	case *ast.DeferStmt:
		// Exempt deferred closures - they're commonly used for cleanup/telemetry
		if funcLit, ok := node.Call.Fun.(*ast.FuncLit); ok {
			exempt[funcLit] = true
		}

	case *ast.GoStmt:
		// Exempt goroutine closures - they need to capture context
		if funcLit, ok := node.Call.Fun.(*ast.FuncLit); ok {
			exempt[funcLit] = true
		}

	case *ast.ReturnStmt:
		// Exempt closures returned from functions (handler factory pattern)
		for _, result := range node.Results {
			if funcLit, ok := result.(*ast.FuncLit); ok {
				exempt[funcLit] = true
			}
		}

	case *ast.CallExpr:
		// Check for visitor pattern callbacks (e.g., ast.Inspect, f.VisitAll)
		if !exemptVisitorFuncs[getCallFuncName(node)] {
			return
		}
		for _, arg := range node.Args {
			if funcLit, ok := arg.(*ast.FuncLit); ok {
				exempt[funcLit] = true
			}
		}

	case *ast.KeyValueExpr:
		// Check for Cobra RunE/Run and HTTP handler fields
		ident, ok := node.Key.(*ast.Ident)
		if !ok || (!exemptCobraFields[ident.Name] && !exemptHTTPFields[ident.Name]) {
			return
		}
		if funcLit, ok := node.Value.(*ast.FuncLit); ok {
			exempt[funcLit] = true
		}
	}
}

func checkClosure(pass *analysis.Pass, reporter *nolint.Reporter, closure *ast.FuncLit) {
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
	captured := countCapturedVars(pass, closure)
	if captured > MaxCapturedVars {
		reporter.Reportf(closure.Pos(),
			"closure captures %d variables from outer scope (max %d); consider passing them as parameters or extracting to a named function",
			captured, MaxCapturedVars)
	}
}

// countStatements counts the statements in block. Blocks themselves (the
// closure's body, an if's braces) only group statements and are not counted.
func countStatements(block *ast.BlockStmt) int {
	count := 0
	ast.Inspect(block, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.BlockStmt:
			// Not a statement of its own; count what it holds
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
	body := nestedBody(node)
	if body == nil {
		return current
	}
	return listNestingDepth(body.List, current)
}

// nestedBody returns the block a nesting statement (or a bare block) opens,
// or nil when node does not open one.
func nestedBody(node ast.Node) *ast.BlockStmt {
	switch n := node.(type) {
	case *ast.BlockStmt:
		return n
	case *ast.IfStmt:
		return n.Body
	case *ast.ForStmt:
		return n.Body
	case *ast.RangeStmt:
		return n.Body
	case *ast.SwitchStmt:
		return n.Body
	case *ast.TypeSwitchStmt:
		return n.Body
	case *ast.SelectStmt:
		return n.Body
	default:
		return nil
	}
}

// stmtNestingDepth returns the nesting depth reached by stmt when it appears
// in a block at depth current.
func stmtNestingDepth(stmt ast.Stmt, current int) int {
	switch s := stmt.(type) {
	case *ast.IfStmt:
		return max(maxNestingDepth(s, current+1), elseNestingDepth(s.Else, current))
	case *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
		return maxNestingDepth(s, current+1)
	case *ast.BlockStmt:
		return maxNestingDepth(s, current)
	case *ast.CaseClause:
		// The switch already opened a level; its cases share it
		return listNestingDepth(s.Body, current)
	case *ast.CommClause:
		return listNestingDepth(s.Body, current)
	case *ast.LabeledStmt:
		return stmtNestingDepth(s.Stmt, current)
	default:
		return current
	}
}

// listNestingDepth returns the deepest nesting reached by a list of statements
// (a block's or a case clause's) at depth current.
func listNestingDepth(stmts []ast.Stmt, current int) int {
	maxDepth := current
	for _, stmt := range stmts {
		maxDepth = max(maxDepth, stmtNestingDepth(stmt, current))
	}
	return maxDepth
}

// elseNestingDepth returns the nesting depth reached by an if statement's
// else branch, which is an else-if or a plain block.
func elseNestingDepth(els ast.Stmt, current int) int {
	switch e := els.(type) {
	case *ast.IfStmt:
		// An else-if sits at the same level as the if it continues, and may
		// have an else of its own
		return stmtNestingDepth(e, current)
	case *ast.BlockStmt:
		return maxNestingDepth(e, current)
	default:
		return current
	}
}

// countCapturedVars counts the distinct local variables closure uses that are
// declared outside it: the enclosing function's parameters and locals, and
// those of any enclosing closure. Package-level variables, struct fields and
// the closure's own parameters and locals are not captures.
func countCapturedVars(pass *analysis.Pass, closure *ast.FuncLit) int {
	captured := make(map[*types.Var]bool)
	ast.Inspect(closure.Body, func(n ast.Node) bool {
		ident, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		v, ok := pass.TypesInfo.Uses[ident].(*types.Var)
		if !ok || v.IsField() || v.Parent() == nil || v.Parent() == pass.Pkg.Scope() {
			return true
		}
		if v.Pos() < closure.Pos() || v.Pos() >= closure.End() {
			captured[v] = true
		}
		return true
	})
	return len(captured)
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
