package interfaceconsistency_test

import (
	"reflect"
	"sort"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/spechtlabs/golint-sl/interfaceconsistency"
)

func TestInterfaceConsistencyAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, interfaceconsistency.Analyzer, "a")
}

func TestInterfaceConsistencyMainPackage(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, interfaceconsistency.Analyzer, "mainpkg")
}

func TestInterfaceConsistencyMockPackage(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, interfaceconsistency.Analyzer, "a/mock")
}

func TestAnalyzeInterfaces(t *testing.T) {
	collector := &analysis.Analyzer{
		Name:     "collectinterfaces",
		Doc:      "collects interface information for tests",
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Run: func(pass *analysis.Pass) (any, error) {
			return interfaceconsistency.AnalyzeInterfaces(pass), nil
		},
		ResultType: reflect.TypeFor[*interfaceconsistency.InterfaceInfo](),
	}

	tests := []struct {
		pkg  string
		want []string
	}{
		{pkg: "mainpkg", want: nil},
		{pkg: "collect", want: []string{"Reader", "writer"}},
	}

	for _, tt := range tests {
		t.Run(tt.pkg, func(t *testing.T) {
			results := analysistest.Run(t, analysistest.TestData(), collector, tt.pkg)
			if len(results) != 1 {
				t.Fatalf("got %d results, want 1", len(results))
			}
			info, ok := results[0].Result.(*interfaceconsistency.InterfaceInfo)
			if !ok {
				t.Fatalf("result has type %T, want *InterfaceInfo", results[0].Result)
			}
			got := append([]string(nil), info.Interfaces...)
			sort.Strings(got)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Interfaces = %v, want %v", got, tt.want)
			}
			if len(info.Implementations) != 0 {
				t.Errorf("Implementations = %v, want empty", info.Implementations)
			}
			if info.MissingMocks != nil {
				t.Errorf("MissingMocks = %v, want nil", info.MissingMocks)
			}
		})
	}
}
