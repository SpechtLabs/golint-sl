// Package returninterface provides an analyzer that enforces "accept interfaces, return structs".
//
// This is a fundamental Go principle: functions should accept interface parameters
// for flexibility, but return concrete types for clarity and usability.
package returninterface

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

const Doc = `enforce "accept interfaces, return structs" principle

Functions should:
- Accept interface parameters for flexibility (dependency injection, testing)
- Return concrete types for clarity (callers know exactly what they get)

Good pattern:
    // Accept interface
    func ProcessReader(r io.Reader) (*Result, error) {
        // Can accept any Reader: files, buffers, HTTP bodies...
        return &Result{...}, nil  // Return concrete type
    }

Bad pattern:
    // Returns interface - caller doesn't know what they get
    func GetStorage() Storage {
        return &FileStorage{}
    }

    // Should be:
    func NewFileStorage() *FileStorage {
        return &FileStorage{}
    }

Exceptions:
- Factory functions that must return different implementations
- Standard library interfaces (io.Reader, error)
- Methods implementing interfaces`

var Analyzer = &analysis.Analyzer{
	Name:     "returninterface",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// Standard library interfaces that are acceptable to return
var acceptableReturnInterfaces = map[string]bool{
	// Error handling
	"error": true,

	// IO interfaces
	"io.Reader":     true,
	"io.Writer":     true,
	"io.Closer":     true,
	"io.ReadCloser": true,
	"io.ReadWriter": true,

	// Context
	"context.Context": true,

	// Common stdlib interfaces
	"fmt.Stringer":   true,
	"sort.Interface": true,

	// HTTP
	"http.Handler":      true,
	"http.RoundTripper": true,
}

// Function name patterns that suggest factory functions (acceptable to return interface)
var factoryPatterns = []string{
	"New",     // NewStorage() Storage
	"Create",  // CreateHandler() Handler
	"Build",   // BuildClient() Client
	"Make",    // MakeProcessor() Processor
	"Get",     // GetInstance() Instance (singleton-like)
	"Open",    // Open() File
	"Connect", // Connect() Connection
}

func run(pass *analysis.Pass) (interface{}, error) {
	reporter := nolint.NewReporter(pass)
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return
		}

		checkFunction(reporter, pass, fn)
	})

	return nil, nil
}

func checkFunction(reporter *nolint.Reporter, pass *analysis.Pass, fn *ast.FuncDecl) {
	if fn.Type.Results == nil {
		return
	}

	// Skip methods implementing interfaces
	if fn.Recv != nil {
		return
	}

	// Skip test files
	filename := pass.Fset.Position(fn.Pos()).Filename
	if strings.HasSuffix(filename, "_test.go") {
		return
	}

	// Check if this looks like a factory function
	if isFactoryFunction(fn.Name.Name) {
		return
	}

	for _, result := range fn.Type.Results.List {
		// Check if return type is an interface
		if isNonAcceptableInterface(pass, result.Type) {
			typeName := types.ExprString(result.Type)
			reporter.Reportf(result.Pos(),
				"function %q returns interface %q; return concrete type instead (\"accept interfaces, return structs\")",
				fn.Name.Name, typeName)
		}
	}
}

func isFactoryFunction(name string) bool {
	lowerName := strings.ToLower(name)
	for _, pattern := range factoryPatterns {
		if strings.HasPrefix(lowerName, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

func isNonAcceptableInterface(pass *analysis.Pass, expr ast.Expr) bool {
	// Get the type
	tv, ok := pass.TypesInfo.Types[expr]
	if !ok {
		// Fallback to AST-based check
		return isInterfaceAST(expr)
	}

	// Check if it's an interface type
	iface, ok := tv.Type.Underlying().(*types.Interface)
	if !ok {
		return false
	}

	// Empty interface is handled by emptyinterface analyzer
	if iface.Empty() {
		return false
	}

	// Check if it's an acceptable interface
	typeName := types.ExprString(expr)
	if acceptableReturnInterfaces[typeName] {
		return false
	}

	// Check common interface names that are acceptable
	// Error interfaces are idiomatic Go - allow both "error" and "Error" suffix
	lowerTypeName := strings.ToLower(typeName)
	if strings.HasSuffix(lowerTypeName, "error") || strings.HasSuffix(lowerTypeName, ".error") {
		return false
	}

	return true
}

func isInterfaceAST(expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.InterfaceType:
		return true

	case *ast.Ident:
		// Could be a named interface type
		// Check common patterns
		name := t.Name
		// Interface names often end with "er" or start with "I"
		if strings.HasSuffix(name, "er") && !strings.HasSuffix(name, "Error") {
			return true
		}
		// Common interface type names
		if name == "any" || name == "error" {
			return false // Handled elsewhere
		}
		return false

	case *ast.SelectorExpr:
		// pkg.Type - check against acceptable list
		typeName := types.ExprString(t)
		return !acceptableReturnInterfaces[typeName]
	}

	return false
}
