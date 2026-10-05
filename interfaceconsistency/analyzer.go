// Package interfaceconsistency provides an analyzer that enforces interface-driven design
// patterns: a package's components depend on each other through interfaces, and
// dependencies are injected rather than created where they are used.
package interfaceconsistency

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

// Doc is the analyzer's documentation.
const Doc = `enforce interface-driven design patterns

This analyzer reports:
1. Struct fields whose name contains a dependency word (Client, Service,
   Repository, Store, Provider, Handler, Resolver, Middleware) and whose type
   is a pointer to a concrete type declared in the same package. Such a field
   should have an interface type, so tests can substitute the dependency.
   Fields with a json tag are data (DTOs, CRDs, API types), not dependencies,
   and are skipped.
2. Calls to a New* function whose name contains one of those words from a
   function that is not itself a constructor: the dependency is created
   where it is used instead of being injected. Test files, main packages
   (the composition root) and NewTest* helpers are exempt.

Constructors are free to return concrete types ("accept interfaces, return
structs"; see the returninterface analyzer). Interface-driven design enables
testability and loose coupling.`

// Analyzer reports code that depends on concrete types where interfaces are expected.
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

func run(pass *analysis.Pass) (any, error) {
	reporter := nolint.NewReporter(pass)
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	isMainPkg := pass.Pkg.Name() == "main"

	nodeFilter := []ast.Node{
		(*ast.TypeSpec)(nil),
		(*ast.FuncDecl)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.TypeSpec:
			if st, ok := node.Type.(*ast.StructType); ok {
				checkStructFieldsUseInterfaces(reporter, pass, node, st)
			}

		case *ast.FuncDecl:
			filename := pass.Fset.Position(node.Pos()).Filename
			isTestFile := strings.HasSuffix(filename, "_test.go")
			checkDependencyInjection(reporter, node, isTestFile, isMainPkg)
		}
	})

	return nil, nil
}

// checkStructFieldsUseInterfaces ensures struct fields that look like dependencies use interfaces
func checkStructFieldsUseInterfaces(reporter *nolint.Reporter, pass *analysis.Pass, ts *ast.TypeSpec, st *ast.StructType) {
	if st.Fields == nil {
		return
	}

	for _, field := range st.Fields.List {
		// Skip fields with json struct tags — these are serializable data types
		// (DTOs, CRDs, API types), not injectable dependencies.
		if hasJSONTag(field) {
			continue
		}

		if !isLocalConcretePointer(pass, field.Type) {
			continue
		}

		for _, name := range field.Names {
			if !isDependencyName(name.Name) {
				continue
			}
			reporter.Reportf(field.Pos(),
				"field %q in struct %q looks like a dependency; consider using an interface type instead of concrete type for better testability",
				name.Name, ts.Name.Name)
		}
	}
}

// isDependencyName reports whether name contains a dependency word. A name
// containing several (ServiceClient) is still one dependency.
func isDependencyName(name string) bool {
	for _, pattern := range shouldBeInterfacePatterns {
		if strings.Contains(name, pattern) {
			return true
		}
	}
	return false
}

// isLocalConcretePointer reports whether expr's type is a pointer to a named
// non-interface type declared in the package under analysis. Pointers to other
// packages' types (*http.Client, *sql.DB) are the usual way to hold those and
// are not reported.
func isLocalConcretePointer(pass *analysis.Pass, expr ast.Expr) bool {
	ptr, ok := types.Unalias(pass.TypesInfo.TypeOf(expr)).(*types.Pointer)
	if !ok {
		return false
	}

	named, ok := types.Unalias(ptr.Elem()).(*types.Named)
	if !ok || named.Obj().Pkg() != pass.Pkg {
		return false
	}

	_, isInterface := named.Underlying().(*types.Interface)
	return !isInterface
}

// hasJSONTag returns true if the field has a `json:` struct tag.
func hasJSONTag(field *ast.Field) bool {
	if field.Tag == nil {
		return false
	}
	return strings.Contains(field.Tag.Value, `json:`)
}

// checkDependencyInjection ensures dependencies are injected, not created internally
func checkDependencyInjection(reporter *nolint.Reporter, fn *ast.FuncDecl, isTestFile bool, isMainPkg bool) {
	if fn.Body == nil {
		return
	}

	// Skip test files - creating mocks/fixtures inline in tests is standard practice
	if isTestFile {
		return
	}

	// Skip main packages - they are the composition root where concrete wiring belongs
	if isMainPkg {
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

		var funcName string
		switch f := call.Fun.(type) {
		case *ast.Ident:
			funcName = f.Name
		case *ast.SelectorExpr:
			funcName = f.Sel.Name
		}

		if strings.HasPrefix(funcName, "New") && !strings.HasPrefix(funcName, "NewTest") && isDependencyName(funcName) {
			reporter.Reportf(call.Pos(),
				"creating %s inside function; consider injecting it as a dependency for better testability",
				funcName)
		}

		return true
	})
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

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.TypeSpec)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
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
