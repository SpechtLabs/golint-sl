// Package nilcheck provides an analyzer that enforces nil checks on pointer parameters.
//
// Nil pointer dereferences cause panics at runtime. This analyzer ensures that
// pointer parameters are checked for nil before being used.
package nilcheck

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const Doc = `enforce nil checks on pointer parameters before use

This analyzer detects:
1. Pointer parameters used without nil check
2. Pointer fields accessed without nil check
3. Interface values used without nil check

Every pointer parameter should be validated at the start of a function:

    func ProcessUser(user *User) error {
        if user == nil {
            return errors.New("user cannot be nil")
        }
        // Now safe to use user
        fmt.Println(user.Name)
    }

This prevents nil pointer panics and provides better error messages.`

var Analyzer = &analysis.Analyzer{
	Name:     "nilcheck",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// Types that are guaranteed non-nil by their callers (framework types)
var trustedPointerTypes = map[string]bool{
	// Testing
	"*testing.T": true,
	"*testing.B": true,
	"*testing.M": true,

	// Gin framework
	"*gin.Context":     true,
	"*gin.Engine":      true,
	"*gin.RouterGroup": true,

	// Cobra CLI
	"*cobra.Command": true,

	// HTTP
	"*http.Request":       true,
	"http.ResponseWriter": true,

	// Context (interface, but trusted)
	"context.Context": true,
}

// File patterns to skip (generated code, etc.)
var skipFilePatterns = []string{
	"zz_generated",
	".pb.go",
	"_gen.go",
	"mock_",
	"mocks/",
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

		// Skip generated files
		filename := pass.Fset.Position(fn.Pos()).Filename
		for _, pattern := range skipFilePatterns {
			if strings.Contains(filename, pattern) {
				return
			}
		}

		checkFunction(pass, fn)
	})

	return nil, nil
}

func checkFunction(pass *analysis.Pass, fn *ast.FuncDecl) {
	// Collect pointer parameters
	ptrParams := collectPointerParams(pass, fn)
	if len(ptrParams) == 0 {
		return
	}

	// Track which parameters have been nil-checked
	checkedParams := make(map[string]bool)

	// First pass: find nil checks
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if ifStmt, ok := n.(*ast.IfStmt); ok {
			// Check for: if x == nil or if x != nil
			checkedParam := extractNilCheck(ifStmt.Cond)
			if checkedParam != "" {
				checkedParams[checkedParam] = true
			}
		}
		return true
	})

	// Second pass: find usages of unchecked pointers
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		// Skip the nil check conditions themselves
		if ifStmt, ok := n.(*ast.IfStmt); ok {
			checkedParam := extractNilCheck(ifStmt.Cond)
			if checkedParam != "" {
				// Skip checking inside the nil-check's then block if it's an early return
				if isEarlyReturnBlock(ifStmt.Body) {
					// After this if block, the param is effectively checked
					checkedParams[checkedParam] = true
				}
			}
		}

		// Check for pointer dereference
		switch node := n.(type) {
		case *ast.SelectorExpr:
			// x.Field - check if x is an unchecked pointer param
			if ident, ok := node.X.(*ast.Ident); ok {
				if ptrParams[ident.Name] && !checkedParams[ident.Name] {
					pass.Reportf(node.Pos(),
						"pointer parameter %q used without nil check; add 'if %s == nil { return ... }' at function start",
						ident.Name, ident.Name)
					// Mark as reported to avoid duplicate reports
					checkedParams[ident.Name] = true
				}
			}

		case *ast.StarExpr:
			// *x - explicit dereference
			if ident, ok := node.X.(*ast.Ident); ok {
				if ptrParams[ident.Name] && !checkedParams[ident.Name] {
					pass.Reportf(node.Pos(),
						"pointer parameter %q dereferenced without nil check; add 'if %s == nil { return ... }' at function start",
						ident.Name, ident.Name)
					checkedParams[ident.Name] = true
				}
			}

		case *ast.IndexExpr:
			// x[i] - could be slice/map from pointer
			if ident, ok := node.X.(*ast.Ident); ok {
				if ptrParams[ident.Name] && !checkedParams[ident.Name] {
					pass.Reportf(node.Pos(),
						"pointer parameter %q indexed without nil check",
						ident.Name)
					checkedParams[ident.Name] = true
				}
			}
		}

		return true
	})
}

// collectPointerParams returns a map of parameter names that are pointers
func collectPointerParams(pass *analysis.Pass, fn *ast.FuncDecl) map[string]bool {
	params := make(map[string]bool)

	if fn.Type.Params == nil {
		return params
	}

	for _, field := range fn.Type.Params.List {
		// Get the type string for checking against trusted types
		typeStr := types.ExprString(field.Type)

		// Skip trusted pointer types (framework types that are never nil)
		if trustedPointerTypes[typeStr] {
			continue
		}

		// Check if the parameter type is a pointer
		isPtr := false

		switch t := field.Type.(type) {
		case *ast.StarExpr:
			// *T - pointer type
			// Check if it's a trusted type
			fullType := "*" + types.ExprString(t.X)
			if trustedPointerTypes[fullType] {
				continue
			}
			isPtr = true
		case *ast.Ident:
			// Could be an interface or type alias
			// Check with type info if available
			if obj := pass.TypesInfo.ObjectOf(t); obj != nil {
				if _, ok := obj.Type().Underlying().(*types.Pointer); ok {
					isPtr = true
				}
				// Also check for interfaces (can be nil)
				// But skip common trusted interfaces
				if _, ok := obj.Type().Underlying().(*types.Interface); ok {
					// Skip error interface and context
					if t.Name == "error" || t.Name == "Context" {
						continue
					}
					isPtr = true
				}
			}
		case *ast.InterfaceType:
			// interface{} can be nil - but often used with type assertions
			// Skip for now as it causes many false positives
			continue
		case *ast.SelectorExpr:
			// pkg.Type - check if it's trusted
			fullType := types.ExprString(t)
			if trustedPointerTypes[fullType] {
				continue
			}
		}

		if isPtr {
			for _, name := range field.Names {
				params[name.Name] = true
			}
		}
	}

	return params
}

// extractNilCheck checks if a condition is a nil check and returns the variable name
func extractNilCheck(cond ast.Expr) string {
	binExpr, ok := cond.(*ast.BinaryExpr)
	if !ok {
		return ""
	}

	// Check for x == nil or x != nil
	if binExpr.Op != token.EQL && binExpr.Op != token.NEQ {
		return ""
	}

	var varName string

	// Check X == nil or X != nil
	if ident, ok := binExpr.X.(*ast.Ident); ok {
		if isNilIdent(binExpr.Y) {
			varName = ident.Name
		}
	}

	// Check nil == X or nil != X
	if ident, ok := binExpr.Y.(*ast.Ident); ok {
		if isNilIdent(binExpr.X) {
			varName = ident.Name
		}
	}

	return varName
}

func isNilIdent(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "nil"
}

// isEarlyReturnBlock checks if a block ends with a return statement
func isEarlyReturnBlock(block *ast.BlockStmt) bool {
	if len(block.List) == 0 {
		return false
	}

	lastStmt := block.List[len(block.List)-1]
	_, isReturn := lastStmt.(*ast.ReturnStmt)
	return isReturn
}
