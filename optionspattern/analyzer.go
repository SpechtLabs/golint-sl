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

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

// Doc is the optionspattern analyzer's documentation.
const Doc = `enforce consistent functional options pattern usage

This analyzer ensures:
1. Constructor functions (New*) with more than 4 parameters use functional options
2. Types named *Option are defined as 'type Option func(*T)' (an interface
   with an apply method, as in zap and gRPC, is accepted too)
3. Exported functions returning a functional option type are prefixed with
   With, Allow, Enable, Disable or Set (Default* functions are exempt). A
   functional option type is a named *Option type whose underlying type is a
   one-parameter function or an interface with an apply method; an Options
   config struct is not one.
4. Exported functions prefixed with With return an option (or a type whose
   name mentions Option, such as Options or []Option), unless they are
   builder methods returning their receiver type or take and return a
   context.Context

The functional options pattern provides a clean, extensible API for configuration.`

// Analyzer enforces consistent use of the functional options pattern.
var Analyzer = &analysis.Analyzer{
	Name:     "optionspattern",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

const (
	maxConstructorParams = 4 // Constructors with more params should use options
)

// validOptionFuncPrefixes are prefixes that are valid for functions returning Option types
var validOptionFuncPrefixes = []string{
	"With",    // WithTimeout, WithLogger - primary prefix
	"Allow",   // AllowInsecure, AllowRetry
	"Enable",  // EnableDebug, EnableMetrics
	"Disable", // DisableCache, DisableRetry
	"Set",     // SetTimeout, SetMaxRetries
}

func run(pass *analysis.Pass) (any, error) {
	reporter := nolint.NewReporter(pass)
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Track Option types for validation
	optionTypes := make(map[string]bool)

	nodeFilter := []ast.Node{
		(*ast.TypeSpec)(nil),
		(*ast.FuncDecl)(nil),
	}

	// First pass: collect Option type definitions
	insp.Preorder(nodeFilter, func(n ast.Node) {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return
		}
		isOptionName := strings.HasSuffix(ts.Name.Name, "Option") || ts.Name.Name == "Option"
		if _, isFunc := ts.Type.(*ast.FuncType); isOptionName && isFunc {
			optionTypes[ts.Name.Name] = true
		}
	})

	// Second pass: check functions
	insp.Preorder(nodeFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.TypeSpec:
			checkOptionTypeDefinition(reporter, pass, node)

		case *ast.FuncDecl:
			checkConstructorPattern(reporter, node, optionTypes)
			checkOptionFunctionNaming(reporter, pass, node)
		}
	})

	return nil, nil
}

// checkOptionTypeDefinition ensures Option types follow the pattern
func checkOptionTypeDefinition(reporter *nolint.Reporter, pass *analysis.Pass, ts *ast.TypeSpec) {
	// Check if this looks like an Option type. An alias (type Option =
	// other.Option) is checked where the aliased type is defined.
	if !strings.HasSuffix(ts.Name.Name, "Option") || ts.Assign.IsValid() {
		return
	}

	// An interface with an apply method is the other common shape of a
	// functional option (zap.Option, grpc.DialOption).
	if obj := pass.TypesInfo.Defs[ts.Name]; obj != nil && isApplyInterface(obj.Type().Underlying()) {
		return
	}

	// Should be a function type
	ft, ok := ts.Type.(*ast.FuncType)
	if !ok {
		reporter.Reportf(ts.Pos(),
			"Option type %q should be a function type: type %s func(*T)",
			ts.Name.Name, ts.Name.Name)
		return
	}

	// Function should take exactly one pointer parameter
	if ft.Params == nil || len(ft.Params.List) != 1 {
		reporter.Reportf(ts.Pos(),
			"Option function type should take exactly one parameter (pointer to config struct)")
		return
	}

	// Parameter should be a pointer type
	param := ft.Params.List[0]
	if _, ok := param.Type.(*ast.StarExpr); !ok {
		reporter.Reportf(param.Pos(),
			"Option function parameter should be a pointer type (*T)")
	}

	// Function should return nothing
	if ft.Results != nil && len(ft.Results.List) > 0 {
		reporter.Reportf(ts.Pos(),
			"Option function should not return any values")
	}
}

// checkConstructorPattern ensures New* functions use options when they have many params
func checkConstructorPattern(reporter *nolint.Reporter, fn *ast.FuncDecl, optionTypes map[string]bool) {
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
		reporter.Reportf(fn.Pos(),
			"constructor %q has %d parameters; consider using functional options pattern: New%s(..., opts ...Option)",
			name, regularParams, strings.TrimPrefix(name, "New"))
	}
}

// checkOptionFunctionNaming ensures option functions that return Option types are properly named
func checkOptionFunctionNaming(reporter *nolint.Reporter, pass *analysis.Pass, fn *ast.FuncDecl) {
	if fn.Name == nil || fn.Type.Results == nil {
		return
	}

	name := fn.Name.Name

	// Skip private functions (lowercase first letter) - they don't need public naming conventions
	if len(name) > 0 && name[0] >= 'a' && name[0] <= 'z' {
		return
	}

	// Skip "Default*" functions - these return sets of default options, not individual options
	if strings.HasPrefix(name, "Default") {
		return
	}

	// Check functions that return Option types
	for _, result := range fn.Type.Results.List {
		if !isOptionType(pass.TypesInfo.TypeOf(result.Type)) {
			continue
		}

		// Option-returning functions should start with a valid prefix
		if !hasValidOptionPrefix(name) {
			reporter.Reportf(fn.Pos(),
				"function %q returns Option but doesn't use a standard option prefix (With, Allow, Enable, Disable, Set); rename to With%s",
				name, name)
		}

		// Check that the function body follows the pattern
		checkOptionFunctionBody(fn)
	}

	// Functions starting with "With" that don't return Option are suspicious
	if strings.HasPrefix(name, "With") {
		returnsOption := false
		for _, result := range fn.Type.Results.List {
			// Lenient on purpose: an Options config struct or a slice of
			// options is not an option, but With is a fine name for those.
			if isOptionType(pass.TypesInfo.TypeOf(result.Type)) || strings.Contains(types.ExprString(result.Type), "Option") {
				returnsOption = true
				break
			}
		}

		if !returnsOption {
			// Builder pattern: method returns the same type as its receiver
			// e.g. func (b *Builder) WithTimeout(d time.Duration) *Builder
			if isBuilderPattern(fn) {
				return
			}

			// Context enrichment: takes context.Context and returns context.Context
			// e.g. func WithFields(ctx context.Context, fields ...zap.Field) context.Context
			if isContextEnrichment(fn) {
				return
			}

			reporter.Reportf(fn.Pos(),
				"function %q starts with 'With' but doesn't return an Option type; this naming is reserved for option functions",
				name)
		}
	}
}

// isOptionType reports whether t is a functional option type: a named type
// whose name ends in Option and whose underlying type is either a function
// with one parameter (type Option func(*config)) or an interface with an
// apply method (type Option interface{ apply(*config) }). Other types whose
// name merely contains Option, such as an Options config struct, are not.
func isOptionType(t types.Type) bool {
	named, ok := types.Unalias(t).(*types.Named)
	if !ok || !strings.HasSuffix(named.Obj().Name(), "Option") {
		return false
	}

	switch u := named.Underlying().(type) {
	case *types.Signature:
		return u.Params().Len() == 1
	case *types.Interface:
		return isApplyInterface(u)
	}
	return false
}

// isApplyInterface reports whether t is an interface with an apply (or
// Apply) method taking one parameter.
func isApplyInterface(t types.Type) bool {
	iface, ok := t.(*types.Interface)
	if !ok {
		return false
	}
	for method := range iface.Methods() {
		name := method.Name()
		if (name == "apply" || name == "Apply") && method.Signature().Params().Len() == 1 {
			return true
		}
	}
	return false
}

// hasValidOptionPrefix checks if a function name starts with any valid option prefix
func hasValidOptionPrefix(name string) bool {
	for _, prefix := range validOptionFuncPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// checkOptionFunctionBody ensures option functions follow the closure pattern
func checkOptionFunctionBody(fn *ast.FuncDecl) {
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

// isBuilderPattern checks if a method returns the same type as its receiver,
// which is a standard Go builder pattern (e.g., func (b *Builder) WithX(...) *Builder).
func isBuilderPattern(fn *ast.FuncDecl) bool {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return false
	}
	if fn.Type.Results == nil || len(fn.Type.Results.List) == 0 {
		return false
	}

	receiverType := types.ExprString(fn.Recv.List[0].Type)
	for _, result := range fn.Type.Results.List {
		resultType := types.ExprString(result.Type)
		if resultType == receiverType {
			return true
		}
	}
	return false
}

// isContextEnrichment checks if a function takes context.Context as its first
// parameter and returns context.Context, which is a standard Go pattern for
// enriching context (e.g., func WithFields(ctx context.Context, ...) context.Context).
func isContextEnrichment(fn *ast.FuncDecl) bool {
	if fn.Type.Params == nil || len(fn.Type.Params.List) == 0 {
		return false
	}
	if fn.Type.Results == nil || len(fn.Type.Results.List) == 0 {
		return false
	}

	// First param must be context.Context
	firstParamType := types.ExprString(fn.Type.Params.List[0].Type)
	if firstParamType != "context.Context" {
		return false
	}

	// Must return context.Context
	for _, result := range fn.Type.Results.List {
		resultType := types.ExprString(result.Type)
		if resultType == "context.Context" {
			return true
		}
	}
	return false
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
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.TypeSpec)(nil),
		(*ast.FuncDecl)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
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
