package sideeffects_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"slices"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"

	"github.com/spechtlabs/golint-sl/internal/nolint"
	"github.com/spechtlabs/golint-sl/sideeffects"
)

// sensitiveLeakAnalyzer wires the exported SSA helpers into an analyzer so
// analysistest can check them against testdata.
var sensitiveLeakAnalyzer = &analysis.Analyzer{
	Name:     "sensitiveleak",
	Doc:      "test analyzer for sideeffects.CheckSensitiveDataLeak",
	Requires: []*analysis.Analyzer{buildssa.Analyzer},
	Run:      runSensitiveLeak,
}

func TestSideEffectsAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, sideeffects.Analyzer, "a")
}

func TestCheckSensitiveDataLeak(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, sensitiveLeakAnalyzer, "leak")
}

func TestTrackDataFlow(t *testing.T) {
	tests := []struct {
		name  string
		src   string
		value func(pkg *ssa.Package) (*ssa.Function, ssa.Value)
		want  int
	}{
		{
			name:  "unused parameter has no flow",
			src:   "package p\nfunc f(s string) {}\n",
			value: param("f"),
			want:  0,
		},
		{
			name:  "parameter passed to a call",
			src:   "package p\nfunc g(string) {}\nfunc f(s string) { g(s) }\n",
			value: param("f"),
			want:  1,
		},
		{
			name:  "parameter flows through a derived value into a call",
			src:   "package p\nfunc g(string) {}\nfunc f(s string) { g(s + \"x\") }\n",
			value: param("f"),
			want:  2,
		},
		{
			name:  "a value that flows around a loop is followed once",
			src:   "package p\nfunc loop(s string) string { for i := 0; i < 3; i++ { s = s + \"x\" }; return s }\n",
			value: param("loop"),
			want:  4,
		},
		{
			name: "globals have no referrers",
			src:  "package p\nvar G int\nfunc f() { G = 1 }\n",
			value: func(pkg *ssa.Package) (*ssa.Function, ssa.Value) {
				return pkg.Func("f"), pkg.Var("G")
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, value := tt.value(buildSSA(t, tt.src))
			if got := sideeffects.TrackDataFlow(fn, value); len(got) != tt.want {
				t.Errorf("TrackDataFlow() returned %d instructions %v, want %d", len(got), got, tt.want)
			}
		})
	}
}

func runSensitiveLeak(pass *analysis.Pass) (any, error) {
	reporter := nolint.NewReporter(pass)
	for _, fn := range sideeffects.GetAllFunctions(sideeffects.GetSSAPackage(pass)) {
		sideeffects.CheckSensitiveDataLeak(reporter, fn, []string{"password", "token"})
	}
	return nil, nil
}

func param(name string) func(pkg *ssa.Package) (*ssa.Function, ssa.Value) {
	return func(pkg *ssa.Package) (*ssa.Function, ssa.Value) {
		fn := pkg.Func(name)
		return fn, fn.Params[0]
	}
}

func buildSSA(t *testing.T, src string) *ssa.Package {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg, _, err := ssautil.BuildPackage(&types.Config{}, fset, types.NewPackage("p", "p"), []*ast.File{f}, ssa.SanityCheckFunctions)
	if err != nil {
		t.Fatal(err)
	}
	return pkg
}

func TestGetAllFunctions(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "functions and methods",
			src:  "package p\ntype T struct{}\nfunc (T) M() {}\nfunc (*T) P() {}\nfunc f() {}\n",
			want: []string{"(*p.T).P", "(p.T).M", "p.f", "p.init"},
		},
		{
			name: "type aliases have no methods of their own",
			src:  "package p\ntype A = int\ntype T struct{}\ntype B = T\nfunc (T) M() {}\n",
			want: []string{"(p.T).M", "p.init"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			for _, fn := range sideeffects.GetAllFunctions(buildSSA(t, tt.src)) {
				got = append(got, fn.String())
			}
			slices.Sort(got)
			if !slices.Equal(got, tt.want) {
				t.Errorf("GetAllFunctions() = %v, want %v", got, tt.want)
			}
		})
	}
}
