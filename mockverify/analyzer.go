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
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const Doc = `enforce compile-time interface verification for mocks

This analyzer ensures that mock implementations include a compile-time
interface verification statement:

    var _ InterfaceName = &MockImplementation{}

This pattern catches interface drift at compile time rather than at runtime,
preventing issues like:
- Mock missing new interface methods
- Interface signature changes breaking tests silently
- Incomplete mock implementations

The analyzer checks files in mock/ directories or files named *_mock.go
and ensures they have the verification pattern.`

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

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Track mock structs and their interface verifications
	mockStructs := make(map[string]bool)       // mock name -> true
	verifiedMocks := make(map[string]bool)     // mock name -> has verification
	mockPositions := make(map[string]ast.Node) // mock name -> position for reporting

	nodeFilter := []ast.Node{
		(*ast.File)(nil),
		(*ast.GenDecl)(nil),
		(*ast.TypeSpec)(nil),
	}

	// Collect all mock structs and interface verifications
	inspect.Preorder(nodeFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.File:
			filename := pass.Fset.Position(node.Pos()).Filename

			// Only check mock files
			if !isMockFile(filename) {
				return
			}

		case *ast.GenDecl:
			// Check for var _ Interface = &Mock{} pattern
			for _, spec := range node.Specs {
				if vs, ok := spec.(*ast.ValueSpec); ok {
					checkInterfaceVerification(vs, verifiedMocks)
				}
			}

		case *ast.TypeSpec:
			// Check if this is a mock struct
			if _, ok := node.Type.(*ast.StructType); ok {
				if isMockName(node.Name.Name) {
					mockStructs[node.Name.Name] = true
					mockPositions[node.Name.Name] = node
				}
			}
		}
	})

	// Report mocks without verification
	for mockName := range mockStructs {
		if !verifiedMocks[mockName] {
			if pos, ok := mockPositions[mockName]; ok {
				pass.Reportf(pos.Pos(),
					"mock %q should have compile-time interface verification: var _ InterfaceName = &%s{}",
					mockName, mockName)
			}
		}
	}

	return nil, nil
}

// isMockFile checks if a file is a mock file based on path or name
func isMockFile(filename string) bool {
	// Check if in mock directory
	dir := filepath.Dir(filename)
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

// isMockName checks if a type name indicates a mock
func isMockName(name string) bool {
	for _, pattern := range MockNamePatterns {
		if strings.Contains(name, pattern) {
			return true
		}
	}
	return false
}

// checkInterfaceVerification checks if a var spec is an interface verification
// Pattern: var _ Interface = &Mock{}
func checkInterfaceVerification(vs *ast.ValueSpec, verifiedMocks map[string]bool) {
	// Must have blank identifier
	if len(vs.Names) != 1 || vs.Names[0].Name != "_" {
		return
	}

	// Must have exactly one value
	if len(vs.Values) != 1 {
		return
	}

	// Value should be &MockType{} or (*MockType)(nil)
	switch v := vs.Values[0].(type) {
	case *ast.UnaryExpr:
		// &Mock{}
		if v.Op.String() == "&" {
			if composite, ok := v.X.(*ast.CompositeLit); ok {
				if ident, ok := composite.Type.(*ast.Ident); ok {
					if isMockName(ident.Name) {
						verifiedMocks[ident.Name] = true
					}
				}
			}
		}

	case *ast.CallExpr:
		// (*Mock)(nil)
		if paren, ok := v.Fun.(*ast.ParenExpr); ok {
			if star, ok := paren.X.(*ast.StarExpr); ok {
				if ident, ok := star.X.(*ast.Ident); ok {
					if isMockName(ident.Name) {
						verifiedMocks[ident.Name] = true
					}
				}
			}
		}
	}
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
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	mockStructs := make(map[string]bool)
	verifiedMocks := make(map[string]bool)

	nodeFilter := []ast.Node{
		(*ast.GenDecl)(nil),
		(*ast.TypeSpec)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.GenDecl:
			for _, spec := range node.Specs {
				if vs, ok := spec.(*ast.ValueSpec); ok {
					checkInterfaceVerification(vs, verifiedMocks)
				}
			}

		case *ast.TypeSpec:
			if _, ok := node.Type.(*ast.StructType); ok {
				if isMockName(node.Name.Name) {
					mockStructs[node.Name.Name] = true
					info.Mocks = append(info.Mocks, node.Name.Name)
				}
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
