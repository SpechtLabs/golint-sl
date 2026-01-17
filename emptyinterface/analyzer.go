// Package emptyinterface provides an analyzer that detects problematic uses of interface{}/any.
//
// The empty interface (interface{} or any) bypasses Go's type system.
// While sometimes necessary, it should be used sparingly and wrapped with type-safe APIs.
package emptyinterface

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

const Doc = `detect problematic uses of interface{}/any

The empty interface bypasses Go's type system and should be used sparingly.
Common problematic patterns:

1. Maps with interface{} values: map[string]interface{}
   - Wrap with type-safe getters/setters
   
2. Slices of interface{}: []interface{}
   - Use concrete types or generics (Go 1.18+)
   
3. Functions returning interface{}
   - Return concrete types; "accept interfaces, return structs"

4. Type assertions without ok check
   - Always use val, ok := x.(Type)

Acceptable uses:
- json.Marshal/Unmarshal (stdlib necessity)
- fmt.Printf and similar (variadic printing)
- Reflection-based code (encoding, ORM)

Example of wrapping unsafe code:
    // Bad: Exposes interface{} to callers
    func Get(key string) interface{} { ... }

    // Good: Type-safe wrapper
    type ItemCache struct { store map[string]interface{} }
    func (c *ItemCache) Get(key string) (Item, error) {
        v, ok := c.store[key]
        if !ok {
            return Item{}, ErrNotFound
        }
        item, ok := v.(Item)
        if !ok {
            return Item{}, ErrInvalidType
        }
        return item, nil
    }`

var Analyzer = &analysis.Analyzer{
	Name:     "emptyinterface",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	reporter := nolint.NewReporter(pass)
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.TypeSpec)(nil),
		(*ast.TypeAssertExpr)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.FuncDecl:
			checkFuncDecl(reporter, node)

		case *ast.TypeSpec:
			checkTypeSpec(reporter, node)

		case *ast.TypeAssertExpr:
			checkTypeAssertion(node)
		}
	})

	return nil, nil
}

func checkFuncDecl(reporter *nolint.Reporter, fn *ast.FuncDecl) {
	// Check return types for interface{}
	if fn.Type.Results != nil {
		for _, field := range fn.Type.Results.List {
			if isEmptyInterface(field.Type) {
				// Allow if function name suggests it's a wrapper/adapter
				if isAllowedFuncName(fn.Name.Name) {
					continue
				}
				reporter.Reportf(field.Pos(),
					"function %q returns interface{}/any; return concrete types instead (\"accept interfaces, return structs\")",
					fn.Name.Name)
			}
		}
	}

	// Check parameters - less strict, but flag map[string]interface{}
	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			if isMapWithEmptyInterface(field.Type) {
				for _, name := range field.Names {
					reporter.Reportf(field.Pos(),
						"parameter %q is map[string]interface{}; consider using a struct or typed map",
						name.Name)
				}
			}
		}
	}
}

func checkTypeSpec(reporter *nolint.Reporter, ts *ast.TypeSpec) {
	// Check struct fields
	structType, ok := ts.Type.(*ast.StructType)
	if !ok {
		return
	}

	for _, field := range structType.Fields.List {
		// Flag map[string]interface{} fields
		if isMapWithEmptyInterface(field.Type) {
			fieldNames := getFieldNames(field)
			reporter.Reportf(field.Pos(),
				"field %q is map[string]interface{}; consider using a typed struct or wrapping with type-safe methods",
				fieldNames)
		}

		// Flag []interface{} fields
		if isSliceOfEmptyInterface(field.Type) {
			fieldNames := getFieldNames(field)
			reporter.Reportf(field.Pos(),
				"field %q is []interface{}; consider using a concrete slice type or generics",
				fieldNames)
		}
	}
}

func checkTypeAssertion(_ *ast.TypeAssertExpr) {
	// Type assertion checking is complex and requires parent context
	// to determine if the ok pattern is used. This is left as a
	// placeholder for future implementation.
	//
	// TODO: Implement proper type assertion checking by analyzing
	// the parent assignment statement.
}

func isEmptyInterface(expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.InterfaceType:
		// interface{}
		return t.Methods == nil || len(t.Methods.List) == 0

	case *ast.Ident:
		// any (Go 1.18+)
		return t.Name == "any"

	case *ast.SelectorExpr:
		// Could be a type alias
		return false
	}

	return false
}

func isMapWithEmptyInterface(expr ast.Expr) bool {
	mapType, ok := expr.(*ast.MapType)
	if !ok {
		return false
	}

	return isEmptyInterface(mapType.Value)
}

func isSliceOfEmptyInterface(expr ast.Expr) bool {
	arrayType, ok := expr.(*ast.ArrayType)
	if !ok {
		return false
	}

	return isEmptyInterface(arrayType.Elt)
}

func isAllowedFuncName(name string) bool {
	// Functions that commonly need to return interface{}
	allowedPrefixes := []string{
		"Marshal", "Unmarshal", "Decode", "Encode",
		"Get", "Load", "Read", // Generic getters in cache/store implementations
		"Parse", "Convert", // Parsing/conversion functions that return different types
		"Wrap", "Value", // Wrapper/value extraction patterns
	}

	// Also allow lowercase versions for private functions
	lowerName := strings.ToLower(name)
	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(name, prefix) || strings.HasPrefix(lowerName, strings.ToLower(prefix)) {
			return true
		}
	}

	return false
}

func getFieldNames(field *ast.Field) string {
	if len(field.Names) == 0 {
		return types.ExprString(field.Type)
	}

	names := make([]string, len(field.Names))
	for i, name := range field.Names {
		names[i] = name.Name
	}
	return strings.Join(names, ", ")
}
