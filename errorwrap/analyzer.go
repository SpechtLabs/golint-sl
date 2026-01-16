// Package errorwrap provides an analyzer that detects bare error returns
// without proper context wrapping.
package errorwrap

import (
	"go/ast"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const Doc = `detect bare error returns without context

This analyzer detects:
1. Returning err directly without wrapping (return err)
2. Error returns in functions with multiple operations where context is lost
3. Error variables returned without adding context about what failed

Errors should be wrapped with context to create a clear error chain:
  return fmt.Errorf("failed to create user: %w", err)
  return humane.Wrap(err, "failed to create user", "check database connection")

Bare error returns make debugging difficult because you lose the stack context.`

var Analyzer = &analysis.Analyzer{
	Name:     "errorwrap",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return
		}

		// Skip test functions
		if fn.Name != nil && strings.HasPrefix(fn.Name.Name, "Test") {
			return
		}

		// Skip very simple functions (1-2 statements)
		if len(fn.Body.List) <= 2 {
			return
		}

		checkFunction(pass, fn)
	})

	return nil, nil
}

func checkFunction(pass *analysis.Pass, fn *ast.FuncDecl) {
	// Track error assignments and their positions
	errorAssignments := make(map[string]token.Pos)
	errorWrapped := make(map[string]bool)

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.AssignStmt:
			checkErrorAssignment(node, errorAssignments)

		case *ast.CallExpr:
			// Check if this is an error wrapping call
			if isErrorWrap(node) {
				// Mark any error variables used as wrapped
				for _, arg := range node.Args {
					if ident, ok := arg.(*ast.Ident); ok {
						errorWrapped[ident.Name] = true
					}
				}
			}

		case *ast.ReturnStmt:
			checkBareErrorReturn(pass, node, fn, errorAssignments, errorWrapped)
		}
		return true
	})
}

func checkErrorAssignment(assign *ast.AssignStmt, errorAssignments map[string]token.Pos) {
	// Look for assignments like: err := someCall()
	for _, lhs := range assign.Lhs {
		ident, ok := lhs.(*ast.Ident)
		if !ok {
			continue
		}

		// Common error variable names
		if ident.Name == "err" || strings.HasSuffix(ident.Name, "Err") || strings.HasSuffix(ident.Name, "Error") {
			errorAssignments[ident.Name] = assign.Pos()
		}
	}
}

func isErrorWrap(call *ast.CallExpr) bool {
	// Check for common wrapping patterns
	switch fn := call.Fun.(type) {
	case *ast.SelectorExpr:
		funcName := fn.Sel.Name

		// fmt.Errorf with %w
		if funcName == "Errorf" {
			// Check if format string contains %w
			if len(call.Args) > 0 {
				if lit, ok := call.Args[0].(*ast.BasicLit); ok {
					if strings.Contains(lit.Value, "%w") {
						return true
					}
				}
			}
		}

		// humane.Wrap, errors.Wrap, pkg/errors.Wrap
		if funcName == "Wrap" || funcName == "Wrapf" || funcName == "WithMessage" {
			return true
		}

		// humane.New creates a new error (not wrap, but acceptable)
		if funcName == "New" {
			if ident, ok := fn.X.(*ast.Ident); ok {
				if ident.Name == "humane" {
					return true
				}
			}
		}

	case *ast.Ident:
		// errors.New is not wrapping but creating new error
		if fn.Name == "Errorf" {
			return true
		}
	}

	return false
}

func checkBareErrorReturn(pass *analysis.Pass, ret *ast.ReturnStmt, fn *ast.FuncDecl, errorAssignments map[string]token.Pos, errorWrapped map[string]bool) {
	if ret.Results == nil {
		return
	}

	for _, result := range ret.Results {
		ident, ok := result.(*ast.Ident)
		if !ok {
			continue
		}

		// Check if this is an error variable
		if _, isError := errorAssignments[ident.Name]; !isError {
			// Also check for common error names not explicitly tracked
			if ident.Name != "err" && !strings.HasSuffix(ident.Name, "Err") {
				continue
			}
		}

		// Skip if nil
		if ident.Name == "nil" {
			continue
		}

		// Skip if already wrapped
		if errorWrapped[ident.Name] {
			continue
		}

		// This is a bare error return
		// Only report if the function has meaningful operations (not just wrapping another call)
		if hasMultipleOperations(fn) {
			pass.Reportf(ret.Pos(),
				"returning error %q without wrapping; add context with fmt.Errorf(\"operation: %%w\", %s) or humane.Wrap()",
				ident.Name, ident.Name)
		}
	}
}

func hasMultipleOperations(fn *ast.FuncDecl) bool {
	if fn.Body == nil {
		return false
	}

	// Count meaningful statements (excluding just returns and error checks)
	meaningful := 0
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			meaningful++
		case *ast.AssignStmt:
			// Assignments that aren't just error checks
			if len(node.Lhs) > 1 || (len(node.Lhs) == 1 && !isErrorIdent(node.Lhs[0])) {
				meaningful++
			}
		}
		return meaningful < 3 // Stop early if we've found enough
	})

	return meaningful >= 2
}

func isErrorIdent(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "err" || strings.HasSuffix(ident.Name, "Err")
}
