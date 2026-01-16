// Package functionsize enforces function size limits with actionable advice.
//
// Long functions are hard to understand, test, and maintain.
// This analyzer provides specific guidance on how to split them.
package functionsize

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

const Doc = `enforce function size limits with refactoring advice

Functions should be small and focused. This analyzer flags functions that are
too long and provides specific advice on how to refactor them.

Guidelines:
- Ideal: 10-30 lines
- Acceptable: 30-80 lines  
- Warning: 80-120 lines (consider splitting)
- Error: 120+ lines (must split)

Long functions often indicate:
1. Multiple responsibilities (extract into separate functions)
2. Deep nesting (use early returns)
3. Repeated patterns (extract helper functions)
4. Complex conditionals (use strategy pattern or lookup tables)`

var Analyzer = &analysis.Analyzer{
	Name:     "functionsize",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

const (
	warnThreshold  = 80  // Lines to trigger warning
	errorThreshold = 120 // Lines to trigger error
)

func run(pass *analysis.Pass) (interface{}, error) {
	reporter := nolint.NewReporter(pass)
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		fn := n.(*ast.FuncDecl)
		if fn.Body == nil {
			return
		}

		// Skip test files for this analyzer
		filename := pass.Fset.Position(fn.Pos()).Filename
		if strings.HasSuffix(filename, "_test.go") {
			return
		}

		// Calculate function length
		startLine := pass.Fset.Position(fn.Body.Lbrace).Line
		endLine := pass.Fset.Position(fn.Body.Rbrace).Line
		lines := endLine - startLine + 1

		if lines < warnThreshold {
			return
		}

		// Analyze function to provide specific advice
		advice := analyzeFunction(fn)

		if lines >= errorThreshold {
			reporter.Reportf(fn.Pos(),
				"function %s is %d lines (max %d); %s",
				fn.Name.Name, lines, errorThreshold, advice)
		} else if lines >= warnThreshold {
			reporter.Reportf(fn.Pos(),
				"function %s is %d lines (recommended max %d); %s",
				fn.Name.Name, lines, warnThreshold, advice)
		}
	})

	return nil, nil
}

func analyzeFunction(fn *ast.FuncDecl) string {
	var suggestions []string

	// Count different statement types
	ifCount := 0
	forCount := 0
	switchCount := 0
	errCheckCount := 0
	maxNesting := 0

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.IfStmt:
			ifCount++
			if isErrCheck(node) {
				errCheckCount++
			}
		case *ast.ForStmt, *ast.RangeStmt:
			forCount++
		case *ast.SwitchStmt, *ast.TypeSwitchStmt:
			switchCount++
		}
		return true
	})

	maxNesting = calculateMaxNesting(fn.Body, 0)

	// Generate specific advice
	if errCheckCount > 5 {
		suggestions = append(suggestions,
			"extract error-prone operations into helper functions")
	}

	if maxNesting > 3 {
		suggestions = append(suggestions,
			"reduce nesting with early returns")
	}

	if forCount > 2 {
		suggestions = append(suggestions,
			"extract loop bodies into separate functions")
	}

	if switchCount > 1 {
		suggestions = append(suggestions,
			"consider using a lookup table or strategy pattern")
	}

	if ifCount > 8 {
		suggestions = append(suggestions,
			"extract conditional logic into well-named helper functions")
	}

	if len(suggestions) == 0 {
		suggestions = append(suggestions,
			"split into smaller, focused functions with descriptive names")
	}

	return strings.Join(suggestions, "; ")
}

func isErrCheck(ifStmt *ast.IfStmt) bool {
	binExpr, ok := ifStmt.Cond.(*ast.BinaryExpr)
	if !ok {
		return false
	}

	if ident, ok := binExpr.X.(*ast.Ident); ok && ident.Name == "err" {
		return true
	}
	if ident, ok := binExpr.Y.(*ast.Ident); ok && ident.Name == "err" {
		return true
	}
	return false
}

func calculateMaxNesting(node ast.Node, current int) int {
	maxDepth := current

	ast.Inspect(node, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt,
			*ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
			depth := current + 1
			if depth > maxDepth {
				maxDepth = depth
			}
		case *ast.FuncLit:
			return false // Don't count nested functions
		}
		return true
	})

	return maxDepth
}
