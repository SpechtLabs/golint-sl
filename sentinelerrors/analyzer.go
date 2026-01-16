// Package sentinelerrors provides an analyzer that enforces use of sentinel errors.
//
// Sentinel errors (package-level error variables) are preferable to inline errors.New()
// because they can be compared with errors.Is() and provide consistent error messages.
package sentinelerrors

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const Doc = `enforce use of sentinel errors over inline errors.New()

Sentinel errors are package-level error variables that can be:
1. Compared with errors.Is()
2. Reused across the codebase
3. Documented and tested

Good pattern:
    // Package-level sentinel errors
    var (
        ErrNotFound     = errors.New("item not found")
        ErrInvalidInput = errors.New("invalid input")
    )

    func GetItem(id string) (Item, error) {
        if id == "" {
            return Item{}, ErrInvalidInput
        }
        item, ok := cache.Get(id)
        if !ok {
            return Item{}, ErrNotFound
        }
        return item, nil
    }

    // Caller can check specific errors
    if errors.Is(err, ErrNotFound) {
        // handle not found
    }

Bad pattern:
    func GetItem(id string) (Item, error) {
        if id == "" {
            return Item{}, errors.New("invalid input")  // Can't be compared!
        }
        item, ok := cache.Get(id)
        if !ok {
            return Item{}, errors.New("item not found") // Duplicate message possible
        }
        return item, nil
    }

Exceptions:
- Wrapping errors with fmt.Errorf and %w
- One-off errors in main() or tests
- Errors with dynamic context (use fmt.Errorf with %w instead)`

var Analyzer = &analysis.Analyzer{
	Name:     "sentinelerrors",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Track which functions are at package level (init, etc.)
	var currentFunc *ast.FuncDecl
	var inTestFile bool

	nodeFilter := []ast.Node{
		(*ast.File)(nil),
		(*ast.FuncDecl)(nil),
		(*ast.CallExpr)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.File:
			filename := pass.Fset.Position(node.Pos()).Filename
			inTestFile = strings.HasSuffix(filename, "_test.go")

		case *ast.FuncDecl:
			currentFunc = node

		case *ast.CallExpr:
			if inTestFile {
				return
			}

			// Skip main function - one-off errors are acceptable
			if currentFunc != nil && currentFunc.Name.Name == "main" {
				return
			}

			checkErrorsNew(pass, node, currentFunc)
		}
	})

	return nil, nil
}

func checkErrorsNew(pass *analysis.Pass, call *ast.CallExpr, currentFunc *ast.FuncDecl) {
	// Check if this is errors.New()
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	pkgIdent, ok := selector.X.(*ast.Ident)
	if !ok {
		return
	}

	// Check for errors.New()
	if pkgIdent.Name == "errors" && selector.Sel.Name == "New" {
		// Check if this is at package level (var declaration) - that's fine
		if isPackageLevelVar(pass, call) {
			return
		}

		// Check if the error message is dynamic (contains variables)
		if len(call.Args) > 0 {
			if hasVariableContent(call.Args[0]) {
				pass.Reportf(call.Pos(),
					"errors.New() with dynamic content; use fmt.Errorf(\"message: %%w\", err) to wrap errors or define a sentinel error")
				return
			}
		}

		funcName := ""
		if currentFunc != nil {
			funcName = currentFunc.Name.Name
		}

		pass.Reportf(call.Pos(),
			"inline errors.New() in function %q; define a package-level sentinel error (var Err... = errors.New(...)) for better error handling with errors.Is()",
			funcName)
	}

	// Also check for fmt.Errorf without %w (not wrapping an error)
	if pkgIdent.Name == "fmt" && selector.Sel.Name == "Errorf" {
		if len(call.Args) > 0 {
			if !containsWrapVerb(call.Args[0]) {
				// This is fmt.Errorf without wrapping - similar to errors.New
				// but often used for formatting. Only flag if it looks like a constant message
				if isLiteralString(call.Args[0]) && len(call.Args) == 1 {
					pass.Reportf(call.Pos(),
						"fmt.Errorf() without %%w verb and no formatting; use a sentinel error or wrap an existing error")
				}
			}
		}
	}
}

func isPackageLevelVar(pass *analysis.Pass, call *ast.CallExpr) bool {
	// Check if this call is in a var declaration at package level
	// This is complex to determine from the call alone
	// For now, we use a heuristic: check if we're in a function
	// If not in a function, it's package level

	// This would need more sophisticated scope analysis
	// For now, we'll rely on the currentFunc check in the caller
	return false
}

func hasVariableContent(expr ast.Expr) bool {
	// Check if the argument contains variable references (not just literals)
	hasVar := false
	ast.Inspect(expr, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.Ident:
			// Check if it's a variable (not a package name or builtin)
			if node.Obj != nil && node.Obj.Kind == ast.Var {
				hasVar = true
				return false
			}
		case *ast.CallExpr:
			// Contains a function call - dynamic
			hasVar = true
			return false
		}
		return true
	})
	return hasVar
}

func containsWrapVerb(expr ast.Expr) bool {
	lit, ok := expr.(*ast.BasicLit)
	if !ok {
		return false
	}

	return strings.Contains(lit.Value, "%w")
}

func isLiteralString(expr ast.Expr) bool {
	_, ok := expr.(*ast.BasicLit)
	return ok
}
