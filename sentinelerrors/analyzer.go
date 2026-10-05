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

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

// Doc is the analyzer's documentation.
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

// Analyzer reports inline errors.New calls that should be package-level sentinel errors.
var Analyzer = &analysis.Analyzer{
	Name:     "sentinelerrors",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	reporter := nolint.NewReporter(pass)
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	for cur := range insp.Root().Preorder((*ast.CallExpr)(nil)) {
		call := cur.Node().(*ast.CallExpr)

		if strings.HasSuffix(pass.Fset.Position(call.Pos()).Filename, "_test.go") {
			continue
		}

		// A call outside every function declaration belongs to a
		// package-level declaration, which is where sentinel errors live.
		funcDecl := enclosingFuncDecl(cur)
		if funcDecl == nil {
			continue
		}

		// Skip main function - one-off errors are acceptable
		if funcDecl.Name.Name == "main" {
			continue
		}

		checkErrorsNew(reporter, call, funcDecl)
	}

	return nil, nil
}

// enclosingFuncDecl returns the function declaration that contains cur, or
// nil when cur is part of a package-level declaration.
func enclosingFuncDecl(cur inspector.Cursor) *ast.FuncDecl {
	for c := range cur.Enclosing((*ast.FuncDecl)(nil)) {
		return c.Node().(*ast.FuncDecl)
	}

	return nil
}

func checkErrorsNew(reporter *nolint.Reporter, call *ast.CallExpr, funcDecl *ast.FuncDecl) {
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
		checkInlineErrorsNew(reporter, call, funcDecl)
	}

	// Also check for fmt.Errorf without %w (not wrapping an error)
	if pkgIdent.Name == "fmt" && selector.Sel.Name == "Errorf" {
		checkUnwrappedErrorf(reporter, call)
	}
}

// checkInlineErrorsNew reports an errors.New() call made inside a function body
func checkInlineErrorsNew(reporter *nolint.Reporter, call *ast.CallExpr, funcDecl *ast.FuncDecl) {
	// Check if the error message is dynamic (contains variables)
	if len(call.Args) > 0 && hasVariableContent(call.Args[0]) {
		reporter.Reportf(call.Pos(),
			"errors.New() with dynamic content; use fmt.Errorf(\"message: %%w\", err) to wrap errors or define a sentinel error")
		return
	}

	reporter.Reportf(call.Pos(),
		"inline errors.New() in function %q; define a package-level sentinel error (var Err... = errors.New(...)) for better error handling with errors.Is()",
		funcDecl.Name.Name)
}

// checkUnwrappedErrorf reports an fmt.Errorf() call with a constant message and no %w verb
func checkUnwrappedErrorf(reporter *nolint.Reporter, call *ast.CallExpr) {
	// This is fmt.Errorf without wrapping - similar to errors.New
	// but often used for formatting. Only flag if it looks like a constant message
	if len(call.Args) > 0 && !containsWrapVerb(call.Args[0]) &&
		isLiteralString(call.Args[0]) && len(call.Args) == 1 {
		reporter.Reportf(call.Pos(),
			"fmt.Errorf() without %%w verb and no formatting; use humane.New(message, advice...) or define a sentinel error")
	}
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
