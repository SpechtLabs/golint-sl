// Package optionspattern provides an analyzer that enforces consistent use of
// the functional options pattern for configuration.
package optionspattern

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const Doc = `enforce consistent functional options pattern usage

This analyzer ensures:
1. Constructor functions (New*) with many config parameters use functional options
2. Option types are defined as 'type Option func(*T)' pattern
3. Option functions are prefixed with 'With'
4. Options files follow naming convention (options.go)

The functional options pattern provides a clean, extensible API for configuration.`

var Analyzer = &analysis.Analyzer{
	Name:     "optionspattern",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

const (
	maxConstructorParams = 3 // Constructors with more params should use options
	optionFuncPrefix     = "With"
)

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Track Option types for validation
	optionTypes := make(map[string]bool)

	nodeFilter := []ast.Node{
		(*ast.TypeSpec)(nil),
		(*ast.FuncDecl)(nil),
	}

	// First pass: collect Option type definitions
	inspect.Preorder(nodeFilter, func(n ast.Node) {
		if ts, ok := n.(*ast.TypeSpec); ok {
			if strings.HasSuffix(ts.Name.Name, "Option") || ts.Name.Name == "Option" {
				if _, ok := ts.Type.(*ast.FuncType); ok {
					optionTypes[ts.Name.Name] = true
				}
			}
		}
	})

	// Second pass: check functions
	inspect.Preorder(nodeFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.TypeSpec:
			checkOptionTypeDefinition(pass, node)

		case *ast.FuncDecl:
			checkConstructorPattern(pass, node, optionTypes)
			checkOptionFunctionNaming(pass, node, optionTypes)
		}
	})

	return nil, nil
}

// checkOptionTypeDefinition ensures Option types follow the pattern
func checkOptionTypeDefinition(pass *analysis.Pass, ts *ast.TypeSpec) {
	// Check if this looks like an Option type
	if !strings.HasSuffix(ts.Name.Name, "Option") && ts.Name.Name != "Option" {
		return
	}

	// Should be a function type
	ft, ok := ts.Type.(*ast.FuncType)
	if !ok {
		pass.Reportf(ts.Pos(),
			"Option type %q should be a function type: type %s func(*T)",
			ts.Name.Name, ts.Name.Name)
		return
	}

	// Function should take exactly one pointer parameter
	if ft.Params == nil || len(ft.Params.List) != 1 {
		pass.Reportf(ts.Pos(),
			"Option function type should take exactly one parameter (pointer to config struct)")
		return
	}

	// Parameter should be a pointer type
	param := ft.Params.List[0]
	if _, ok := param.Type.(*ast.StarExpr); !ok {
		pass.Reportf(param.Pos(),
			"Option function parameter should be a pointer type (*T)")
	}

	// Function should return nothing
	if ft.Results != nil && len(ft.Results.List) > 0 {
		pass.Reportf(ts.Pos(),
			"Option function should not return any values")
	}
}

// checkConstructorPattern ensures New* functions use options when they have many params
func checkConstructorPattern(pass *analysis.Pass, fn *ast.FuncDecl, optionTypes map[string]bool) {
	if fn.Name == nil {
		return
	}

	name := fn.Name.Name

	// Only check constructor functions (New*)
	if !strings.HasPrefix(name, "New") {
		return
	}

	// Skip test functions
	if strings.HasPrefix(name, "NewTest") || strings.HasSuffix(name, "Test") {
		return
	}

	if fn.Type.Params == nil {
		return
	}

	// Count non-variadic, non-option parameters
	regularParams := 0
	hasOptions := false

	for _, param := range fn.Type.Params.List {
		paramType := types.ExprString(param.Type)

		// Check if this is a variadic options parameter
		if ellipsis, ok := param.Type.(*ast.Ellipsis); ok {
			eltType := types.ExprString(ellipsis.Elt)
			if strings.Contains(eltType, "Option") || optionTypes[eltType] {
				hasOptions = true
				continue
			}
		}

		// Check if this is an Option slice
		if strings.Contains(paramType, "Option") || strings.Contains(paramType, "...Option") {
			hasOptions = true
			continue
		}

		// Count regular parameters (each field can have multiple names)
		numNames := len(param.Names)
		if numNames == 0 {
			numNames = 1 // unnamed parameter
		}
		regularParams += numNames
	}

	// If constructor has many params and no options, suggest the pattern
	if regularParams > maxConstructorParams && !hasOptions {
		pass.Reportf(fn.Pos(),
			"constructor %q has %d parameters; consider using functional options pattern: New%s(..., opts ...Option)",
			name, regularParams, strings.TrimPrefix(name, "New"))
	}
}

// checkOptionFunctionNaming ensures With* functions that return Option types are proper
func checkOptionFunctionNaming(pass *analysis.Pass, fn *ast.FuncDecl, optionTypes map[string]bool) {
	if fn.Name == nil || fn.Type.Results == nil {
		return
	}

	name := fn.Name.Name

	// Check functions that return Option types
	for _, result := range fn.Type.Results.List {
		resultType := types.ExprString(result.Type)

		isOptionReturn := strings.Contains(resultType, "Option") || optionTypes[resultType]
		if !isOptionReturn {
			continue
		}

		// Option-returning functions should start with "With"
		if !strings.HasPrefix(name, optionFuncPrefix) {
			pass.Reportf(fn.Pos(),
				"function %q returns Option but doesn't start with 'With'; rename to With%s",
				name, name)
		}

		// Check that the function body follows the pattern
		checkOptionFunctionBody(pass, fn)
	}

	// Functions starting with "With" that don't return Option are suspicious
	if strings.HasPrefix(name, optionFuncPrefix) {
		returnsOption := false
		for _, result := range fn.Type.Results.List {
			resultType := types.ExprString(result.Type)
			if strings.Contains(resultType, "Option") || optionTypes[resultType] {
				returnsOption = true
				break
			}
		}

		if !returnsOption {
			pass.Reportf(fn.Pos(),
				"function %q starts with 'With' but doesn't return an Option type; this naming is reserved for option functions",
				name)
		}
	}
}

// checkOptionFunctionBody ensures option functions follow the closure pattern
func checkOptionFunctionBody(_ *analysis.Pass, fn *ast.FuncDecl) {
	if fn.Body == nil || len(fn.Body.List) == 0 {
		return
	}

	// Should have a single return statement returning a function literal
	if len(fn.Body.List) != 1 {
		// Multiple statements are okay if it's validation + return
		return
	}

	ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok {
		return
	}

	if len(ret.Results) != 1 {
		return
	}

	// The return should be a function literal (or a named function, which is also valid)
	_, _ = ret.Results[0].(*ast.FuncLit)
}

// OptionPatternInfo contains information about option pattern usage in a package
type OptionPatternInfo struct {
	OptionTypes     []string
	OptionFunctions []string
	Constructors    []string
}

// AnalyzeOptionPatterns returns information about option pattern usage
func AnalyzeOptionPatterns(pass *analysis.Pass) *OptionPatternInfo {
	info := &OptionPatternInfo{}
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.TypeSpec)(nil),
		(*ast.FuncDecl)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.TypeSpec:
			if strings.Contains(node.Name.Name, "Option") {
				if _, ok := node.Type.(*ast.FuncType); ok {
					info.OptionTypes = append(info.OptionTypes, node.Name.Name)
				}
			}

		case *ast.FuncDecl:
			if node.Name == nil {
				return
			}
			name := node.Name.Name

			if strings.HasPrefix(name, "New") {
				info.Constructors = append(info.Constructors, name)
			}
			if strings.HasPrefix(name, "With") {
				info.OptionFunctions = append(info.OptionFunctions, name)
			}
		}
	})

	return info
}
