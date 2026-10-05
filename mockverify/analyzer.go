// Package mockverify provides an analyzer that enforces compile-time interface verification
// for mock implementations.
//
// Inspired by the compute-blade-agent pattern:
//
//	var _ ComputeBladeHal = &ComputeBladeHalMock{}
//
// This ensures mocks always implement their interface, catching breaking changes at compile time.
package mockverify

import (
	"go/ast"
	"go/types"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

// Doc is the analyzer's documentation.
const Doc = `enforce compile-time interface verification for mocks

This analyzer ensures that mock implementations include a compile-time
interface verification statement:

    var _ InterfaceName = &MockImplementation{}

This pattern catches interface drift at compile time rather than at runtime,
preventing issues like:
- Mock missing new interface methods
- Interface signature changes breaking tests silently
- Incomplete mock implementations

The analyzer checks the package-level struct types whose name contains Mock,
Fake or Stub as a word (MockStore, StoreFake, but not Stubborn), declared in
files in a mock/ directory or in files named *_mock.go or mock_*.go. Any
blank-identifier declaration in the package whose value has the mock's type
or a pointer to it counts as the verification: &Mock{}, Mock{}, new(Mock)
and (*Mock)(nil) all do.`

// Analyzer reports mock types that lack a compile-time interface assertion.
var Analyzer = &analysis.Analyzer{
	Name:     "mockverify",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// MockNamePatterns are patterns that indicate a mock type
var MockNamePatterns = []string{
	"Mock",
	"Fake",
	"Stub",
}

func run(pass *analysis.Pass) (any, error) {
	reporter := nolint.NewReporter(pass)
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// The verification may live in any file of the package; the mock itself
	// is only checked when a mock file declares it.
	verifiedMocks := make(map[string]bool)
	insp.Preorder([]ast.Node{(*ast.ValueSpec)(nil)}, func(n ast.Node) {
		checkInterfaceVerification(pass, n.(*ast.ValueSpec), verifiedMocks)
	})

	for _, file := range pass.Files {
		if isMockFile(pass.Fset.Position(file.Pos()).Filename) {
			reportUnverifiedMocks(reporter, file, verifiedMocks)
		}
	}

	return nil, nil
}

// reportUnverifiedMocks reports the package-level mock structs file declares
// that no verification in the package covers.
func reportUnverifiedMocks(reporter *nolint.Reporter, file *ast.File, verifiedMocks map[string]bool) {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || !isMockStruct(ts) || verifiedMocks[ts.Name.Name] {
				continue
			}
			reporter.Reportf(ts.Pos(),
				"mock %q should have compile-time interface verification: var _ InterfaceName = &%s{}",
				ts.Name.Name, ts.Name.Name)
		}
	}
}

// isMockFile checks if a file is a mock file based on path or name
func isMockFile(filename string) bool {
	filename = filepath.ToSlash(filename)

	// Check if in mock directory
	dir := filepath.ToSlash(filepath.Dir(filename))
	if strings.HasSuffix(dir, "/mock") || strings.Contains(dir, "/mock/") {
		return true
	}

	// Check if file is named *_mock.go or mock_*.go
	base := filepath.Base(filename)
	if strings.HasSuffix(base, "_mock.go") || strings.HasPrefix(base, "mock_") {
		return true
	}

	return false
}

// isMockStruct reports whether ts declares a struct type with a mock name.
func isMockStruct(ts *ast.TypeSpec) bool {
	_, ok := ts.Type.(*ast.StructType)
	return ok && isMockName(ts.Name.Name)
}

// isMockName reports whether one of MockNamePatterns is a whole word of the
// camel-case name: MockStore and StoreFake are mock names, Stubborn and
// Mockingbird are not.
func isMockName(name string) bool {
	for _, pattern := range MockNamePatterns {
		for i := 0; i < len(name); {
			idx := strings.Index(name[i:], pattern)
			if idx < 0 {
				break
			}
			end := i + idx + len(pattern)
			if end == len(name) {
				return true
			}
			if next, _ := utf8.DecodeRuneInString(name[end:]); !unicode.IsLower(next) {
				return true
			}
			i = end
		}
	}
	return false
}

// checkInterfaceVerification checks if a var spec is an interface verification
// Pattern: var _ Interface = &Mock{}
func checkInterfaceVerification(pass *analysis.Pass, vs *ast.ValueSpec, verifiedMocks map[string]bool) {
	// Must have blank identifier
	if len(vs.Names) != 1 || vs.Names[0].Name != "_" {
		return
	}

	// Must have exactly one value
	if len(vs.Values) != 1 {
		return
	}

	// The value's type decides, so &Mock{}, Mock{}, new(Mock), (*Mock)(nil)
	// and instances of generic mocks such as &Mock[int]{} all count.
	if name, ok := verifiedMockName(pass, vs.Values[0]); ok {
		verifiedMocks[name] = true
	}
}

// verifiedMockName returns the name of the mock type declared in this package
// that value has, directly or through a pointer.
func verifiedMockName(pass *analysis.Pass, value ast.Expr) (string, bool) {
	t := pass.TypesInfo.TypeOf(value)
	if t == nil {
		return "", false
	}
	if ptr, ok := types.Unalias(t).(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return "", false
	}

	// Obj of an instantiated generic type is the generic declaration's.
	obj := named.Obj()
	if obj.Pkg() != pass.Pkg || !isMockName(obj.Name()) {
		return "", false
	}
	return obj.Name(), true
}

// MockInfo contains information about mocks in a package
type MockInfo struct {
	Mocks           []string
	VerifiedMocks   []string
	UnverifiedMocks []string
}

// AnalyzeMocks returns information about mock patterns in the package
func AnalyzeMocks(pass *analysis.Pass) *MockInfo {
	info := &MockInfo{}
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	mockStructs := make(map[string]bool)
	verifiedMocks := make(map[string]bool)

	nodeFilter := []ast.Node{
		(*ast.ValueSpec)(nil),
		(*ast.TypeSpec)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.ValueSpec:
			checkInterfaceVerification(pass, node, verifiedMocks)

		case *ast.TypeSpec:
			if isMockStruct(node) {
				mockStructs[node.Name.Name] = true
				info.Mocks = append(info.Mocks, node.Name.Name)
			}
		}
	})

	for mock := range mockStructs {
		if verifiedMocks[mock] {
			info.VerifiedMocks = append(info.VerifiedMocks, mock)
		} else {
			info.UnverifiedMocks = append(info.UnverifiedMocks, mock)
		}
	}

	return info
}
