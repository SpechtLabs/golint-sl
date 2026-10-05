package humaneerror_test

import (
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/spechtlabs/golint-sl/humaneerror"
)

func TestHumaneErrorAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, humaneerror.Analyzer, "a")
}

func TestHumaneErrorAnalyzerImportAlias(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, humaneerror.Analyzer, "b")
}

// TestHumaneErrorAnalyzerConcurrentPackages analyzes a package of framework
// callbacks and a package without any side by side, repeatedly. The
// checker analyzes packages in parallel, so state shared between passes
// would let the callback exemption of one package suppress diagnostics in
// the other (and the race detector would flag it).
func TestHumaneErrorAnalyzerConcurrentPackages(t *testing.T) {
	testdata := analysistest.TestData()
	for range 5 {
		analysistest.Run(t, testdata, humaneerror.Analyzer, "concurrent/callbacks", "concurrent/plain")
	}
}

func TestIsHumaneErrorType(t *testing.T) {
	named := func(pkgPath, name string) types.Type {
		var pkg *types.Package
		if pkgPath != "" {
			pkg = types.NewPackage(pkgPath, "p")
		}
		obj := types.NewTypeName(token.NoPos, pkg, name, nil)
		return types.NewNamed(obj, types.NewInterfaceType(nil, nil), nil)
	}

	tests := []struct {
		name string
		typ  types.Type
		want bool
	}{
		{name: "nil type", typ: nil, want: false},
		{name: "basic type", typ: types.Typ[types.Int], want: false},
		{name: "universe error has no package", typ: types.Universe.Lookup("error").Type(), want: false},
		{name: "humane Error", typ: named("github.com/sierrasoftworks/humane-errors-go", "Error"), want: true},
		{name: "humane package, other type", typ: named("github.com/sierrasoftworks/humane-errors-go", "Option"), want: false},
		{name: "Error type in another package", typ: named("example.com/errs", "Error"), want: false},
		{name: "pointer to humane Error", typ: types.NewPointer(named("github.com/sierrasoftworks/humane-errors-go", "Error")), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := humaneerror.IsHumaneErrorType(tt.typ); got != tt.want {
				t.Errorf("IsHumaneErrorType(%v) = %v, want %v", tt.typ, got, tt.want)
			}
		})
	}
}
