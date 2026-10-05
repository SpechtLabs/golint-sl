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

// Doc is the emptyinterface analyzer's documentation.
const Doc = `detect problematic uses of interface{}/any

The empty interface bypasses Go's type system and should be used sparingly.
Common problematic patterns:

1. Maps with interface{} values: map[string]interface{}
   - Wrap with type-safe getters/setters
   
2. Slices of interface{}: []interface{}
   - Use concrete types or generics (Go 1.18+)
   
3. Functions returning interface{}
   - Return concrete types; "accept interfaces, return structs"

The analyzer reports results of type interface{} or any (except for functions
whose names mark them as decoders, getters or wrappers, such as Unmarshal,
Get or Value), parameters that are maps with interface{} values, and struct
fields that are maps, slices or arrays of interface{}. It doesn't check type
assertions; errcheck's check-type-assertions setting reports the ones without
an ok check.

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

// Analyzer reports problematic uses of interface{} and any.
var Analyzer = &analysis.Analyzer{
	Name:     "emptyinterface",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	reporter := nolint.NewReporter(pass)
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.TypeSpec)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.FuncDecl:
			checkFuncDecl(pass, reporter, node)

		case *ast.TypeSpec:
			checkTypeSpec(pass, reporter, node)
		}
	})

	return nil, nil
}

func checkFuncDecl(pass *analysis.Pass, reporter *nolint.Reporter, fn *ast.FuncDecl) {
	// Check return types for interface{}
	if fn.Type.Results != nil {
		for _, field := range fn.Type.Results.List {
			// Allow if function name suggests it's a wrapper/adapter
			if !isEmptyInterface(pass, field.Type) || isAllowedFuncName(fn.Name.Name) {
				continue
			}
			reporter.Reportf(field.Pos(),
				"function %q returns interface{}/any; return concrete types instead (\"accept interfaces, return structs\")",
				fn.Name.Name)
		}
	}

	// Check parameters - less strict, but flag maps with interface{} values
	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			if !isMapWithEmptyInterface(pass, field.Type) {
				continue
			}
			for _, name := range field.Names {
				reporter.Reportf(field.Pos(),
					"parameter %q is %s; consider using a struct or typed map",
					name.Name, types.ExprString(field.Type))
			}
		}
	}
}

func checkTypeSpec(pass *analysis.Pass, reporter *nolint.Reporter, ts *ast.TypeSpec) {
	// Check struct fields
	structType, ok := ts.Type.(*ast.StructType)
	if !ok {
		return
	}

	for _, field := range structType.Fields.List {
		// Flag fields holding maps with interface{} values
		if isMapWithEmptyInterface(pass, field.Type) {
			reporter.Reportf(field.Pos(),
				"field %q is %s; consider using a typed struct or wrapping with type-safe methods",
				getFieldNames(field), types.ExprString(field.Type))
		}

		// Flag fields holding slices or arrays of interface{}
		if isSliceOfEmptyInterface(pass, field.Type) {
			reporter.Reportf(field.Pos(),
				"field %q is %s; consider using a concrete element type or generics",
				getFieldNames(field), types.ExprString(field.Type))
		}
	}
}

// isEmptyInterface reports whether expr denotes interface{}, any, or an alias
// of either. A defined type such as "type Value interface{}" is a deliberate
// name for the empty interface and isn't reported.
func isEmptyInterface(pass *analysis.Pass, expr ast.Expr) bool {
	iface, ok := types.Unalias(pass.TypesInfo.TypeOf(expr)).(*types.Interface)
	return ok && iface.Empty()
}

func isMapWithEmptyInterface(pass *analysis.Pass, expr ast.Expr) bool {
	mapType, ok := expr.(*ast.MapType)
	if !ok {
		return false
	}

	return isEmptyInterface(pass, mapType.Value)
}

// isSliceOfEmptyInterface reports whether expr is a slice or an array of
// interface{}.
func isSliceOfEmptyInterface(pass *analysis.Pass, expr ast.Expr) bool {
	arrayType, ok := expr.(*ast.ArrayType)
	if !ok {
		return false
	}

	return isEmptyInterface(pass, arrayType.Elt)
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
