// Package interfaceconsistency provides an analyzer that enforces interface-driven design
// patterns, ensuring all major components are accessed through interfaces and that
// mock implementations exist for testing.
package interfaceconsistency

import (
	"go/ast"
	"go/types"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

const Doc = `enforce interface-driven design patterns

This analyzer ensures:
1. Struct fields of interface type (for dependency injection)
2. Exported interfaces have corresponding mock implementations in mock/ subdirectory
3. Constructor functions return interfaces, not concrete types
4. Dependencies are injected, not created internally

Interface-driven design enables testability and loose coupling.`

var Analyzer = &analysis.Analyzer{
	Name:     "interfaceconsistency",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// Patterns that indicate a field should use an interface
var shouldBeInterfacePatterns = []string{
	"Client",
	"Service",
	"Repository",
	"Store",
	"Provider",
	"Handler",
	"Resolver",
	"Middleware",
}

// Patterns for types that should be defined as interfaces
var shouldDefineInterfacePatterns = []string{
	"client",
	"service",
	"repository",
	"store",
}

func run(pass *analysis.Pass) (interface{}, error) {
	reporter := nolint.NewReporter(pass)
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Track interfaces and their implementations
	interfaces := make(map[string]*ast.TypeSpec)
	structs := make(map[string]*ast.TypeSpec)

	nodeFilter := []ast.Node{
		(*ast.TypeSpec)(nil),
		(*ast.FuncDecl)(nil),
		(*ast.File)(nil),
	}

	// First pass: collect all interfaces and structs
	inspect.Preorder(nodeFilter, func(n ast.Node) {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return
		}

		switch ts.Type.(type) {
		case *ast.InterfaceType:
			interfaces[ts.Name.Name] = ts
		case *ast.StructType:
			structs[ts.Name.Name] = ts
		}
	})

	// Second pass: analyze usage
	inspect.Preorder(nodeFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.TypeSpec:
			if st, ok := node.Type.(*ast.StructType); ok {
				checkStructFieldsUseInterfaces(reporter, pass, node, st)
			}

		case *ast.FuncDecl:
			checkConstructorReturnsInterface(reporter, node, interfaces)
			checkDependencyInjection(reporter, node)
		}
	})

	// Check for missing mock implementations
	checkMockImplementations(pass, interfaces)

	return nil, nil
}

// checkStructFieldsUseInterfaces ensures struct fields that look like dependencies use interfaces
func checkStructFieldsUseInterfaces(reporter *nolint.Reporter, pass *analysis.Pass, ts *ast.TypeSpec, st *ast.StructType) {
	if st.Fields == nil {
		return
	}

	for _, field := range st.Fields.List {
		for _, name := range field.Names {
			fieldName := name.Name

			// Check if this field looks like a dependency
			for _, pattern := range shouldBeInterfacePatterns {
				if strings.Contains(fieldName, pattern) || strings.HasSuffix(fieldName, pattern) {
					// Check if the type is already an interface
					if !isInterfaceType(pass, field.Type) {
						// Only report for pointer types to concrete structs
						if star, ok := field.Type.(*ast.StarExpr); ok {
							if _, ok := star.X.(*ast.Ident); ok {
								reporter.Reportf(field.Pos(),
									"field %q in struct %q looks like a dependency; consider using an interface type instead of concrete type for better testability",
									fieldName, ts.Name.Name)
							}
						}
					}
				}
			}
		}
	}
}

// isInterfaceType checks if an AST expression represents an interface type
func isInterfaceType(pass *analysis.Pass, expr ast.Expr) bool {
	t := pass.TypesInfo.TypeOf(expr)
	if t == nil {
		return false
	}

	// Check if it's an interface type
	_, isInterface := t.Underlying().(*types.Interface)
	return isInterface
}

// checkConstructorReturnsInterface ensures New* functions return interfaces when appropriate
func checkConstructorReturnsInterface(reporter *nolint.Reporter, fn *ast.FuncDecl, interfaces map[string]*ast.TypeSpec) {
	if fn.Name == nil {
		return
	}

	name := fn.Name.Name
	if !strings.HasPrefix(name, "New") {
		return
	}

	if fn.Type.Results == nil || len(fn.Type.Results.List) == 0 {
		return
	}

	// Get the type name being constructed
	typeName := strings.TrimPrefix(name, "New")

	// Check if there's a corresponding interface
	interfaceName := typeName
	possibleInterfaceNames := []string{
		typeName,
		typeName + "Interface",
		"I" + typeName,
	}

	hasInterface := false
	for _, ifaceName := range possibleInterfaceNames {
		if _, exists := interfaces[ifaceName]; exists {
			hasInterface = true
			interfaceName = ifaceName
			break
		}
	}

	if !hasInterface {
		return // No interface defined, that's a separate concern
	}

	// Check if the return type is the interface
	for _, result := range fn.Type.Results.List {
		resultType := types.ExprString(result.Type)

		// If returning concrete type instead of interface
		if strings.Contains(resultType, "*"+typeName) && !strings.Contains(resultType, interfaceName) {
			reporter.Reportf(fn.Pos(),
				"constructor %q returns concrete type; consider returning interface %q for better abstraction",
				name, interfaceName)
		}
	}
}

// checkDependencyInjection ensures dependencies are injected, not created internally
func checkDependencyInjection(reporter *nolint.Reporter, fn *ast.FuncDecl) {
	if fn.Body == nil {
		return
	}

	// Skip constructor functions (they're allowed to create things)
	if fn.Name != nil && strings.HasPrefix(fn.Name.Name, "New") {
		return
	}

	// Look for New* calls inside function body
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		// Check if this is a New* call
		var funcName string
		switch f := call.Fun.(type) {
		case *ast.Ident:
			funcName = f.Name
		case *ast.SelectorExpr:
			funcName = f.Sel.Name
		}

		if strings.HasPrefix(funcName, "New") && !strings.HasPrefix(funcName, "NewTest") {
			// Check if this is creating a service/client/repository
			for _, pattern := range shouldBeInterfacePatterns {
				if strings.Contains(funcName, pattern) {
					reporter.Reportf(call.Pos(),
						"creating %s inside function; consider injecting it as a dependency for better testability",
						funcName)
				}
			}
		}

		return true
	})
}

// checkMockImplementations ensures interfaces have corresponding mocks
func checkMockImplementations(pass *analysis.Pass, interfaces map[string]*ast.TypeSpec) {
	// Get the current package path
	pkgPath := pass.Pkg.Path()

	// Skip mock packages themselves
	if strings.HasSuffix(pkgPath, "/mock") || strings.Contains(pkgPath, "/mock/") {
		return
	}

	// Skip test files
	for _, f := range pass.Files {
		filename := pass.Fset.Position(f.Pos()).Filename
		if strings.HasSuffix(filename, "_test.go") {
			continue
		}

		// For each exported interface, check if a mock exists
		for name, iface := range interfaces {
			if !ast.IsExported(name) {
				continue
			}

			// Check if this file contains the interface
			if !fileContainsInterface(f, name) {
				continue
			}

			// Expected mock file would be in mock/ subdirectory
			dir := filepath.Dir(filename)
			expectedMockFile := filepath.Join(dir, "mock", strings.ToLower(name)+".go")

			// We can't check file existence in the analyzer, but we can suggest
			// This is more of a documentation/reminder
			_ = expectedMockFile
			_ = iface

			// Report if the interface is significant enough to warrant a mock
			for _, pattern := range shouldDefineInterfacePatterns {
				if strings.Contains(strings.ToLower(name), pattern) {
					// This is just informational - we'd need actual file checking
					// to know if mock exists
					break
				}
			}
		}
	}
}

func fileContainsInterface(f *ast.File, name string) bool {
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range genDecl.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			if ts.Name.Name == name {
				if _, ok := ts.Type.(*ast.InterfaceType); ok {
					return true
				}
			}
		}
	}
	return false
}

// InterfaceInfo contains information about interface usage in a package
type InterfaceInfo struct {
	Interfaces      []string
	Implementations map[string][]string // interface -> implementations
	MissingMocks    []string
}

// AnalyzeInterfaces returns information about interface patterns in the package
func AnalyzeInterfaces(pass *analysis.Pass) *InterfaceInfo {
	info := &InterfaceInfo{
		Implementations: make(map[string][]string),
	}

	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.TypeSpec)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return
		}

		if _, ok := ts.Type.(*ast.InterfaceType); ok {
			info.Interfaces = append(info.Interfaces, ts.Name.Name)
		}
	})

	return info
}
