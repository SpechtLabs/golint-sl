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

// Doc is the returninterface analyzer's documentation.
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

// Analyzer enforces the "accept interfaces, return structs" principle.
var Analyzer = &analysis.Analyzer{
	Name:     "returninterface",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// Interfaces that are acceptable to return, keyed by the package path and
// the name of the type, so the key doesn't depend on the import's name.
var acceptableReturnInterfaces = map[string]bool{
	// Error handling
	"error": true,
	"github.com/sierrasoftworks/humane-errors-go.Error": true, // humane-errors-go error type

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
	"net/http.Handler":      true,
	"net/http.RoundTripper": true,
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

func run(pass *analysis.Pass) (any, error) {
	reporter := nolint.NewReporter(pass)
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		checkFunction(reporter, pass, n.(*ast.FuncDecl))
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
	typ := types.Unalias(pass.TypesInfo.TypeOf(expr))

	// A type parameter's underlying type is its constraint, which is an
	// interface, but the caller gets the concrete type it instantiates T with.
	if _, ok := typ.(*types.TypeParam); ok {
		return false
	}

	// Check if it's an interface type
	iface, ok := typ.Underlying().(*types.Interface)
	if !ok {
		return false
	}

	// Empty interface is handled by emptyinterface analyzer
	if iface.Empty() {
		return false
	}

	// An interface literal has no name to allow.
	named, ok := typ.(*types.Named)
	if !ok {
		return true
	}

	obj := named.Obj()
	if acceptableReturnInterfaces[qualifiedName(obj)] {
		return false
	}

	// Error interfaces are idiomatic Go - allow both "error" and "Error" suffix
	return !strings.HasSuffix(strings.ToLower(obj.Name()), "error")
}

// qualifiedName returns the package path and the name of obj, or only the
// name for a predeclared type such as error.
func qualifiedName(obj *types.TypeName) string {
	if obj.Pkg() == nil {
		return obj.Name()
	}

	return obj.Pkg().Path() + "." + obj.Name()
}
