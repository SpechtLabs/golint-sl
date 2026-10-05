package contextpropagation_test

import (
	"reflect"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/spechtlabs/golint-sl/contextpropagation"
)

func TestContextPropagationAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, contextpropagation.Analyzer, "a", "mockdb")
}

// statsAnalyzer exposes AnalyzeContextPropagation as an analyzer result so it
// can run under analysistest.
var statsAnalyzer = &analysis.Analyzer{
	Name:       "contextpropagationstats",
	Doc:        "test wrapper around contextpropagation.AnalyzeContextPropagation",
	Requires:   []*analysis.Analyzer{inspect.Analyzer},
	ResultType: reflect.TypeFor[*contextpropagation.ContextPropagationInfo](),
	Run: func(pass *analysis.Pass) (any, error) {
		return contextpropagation.AnalyzeContextPropagation(pass), nil
	},
}

func TestAnalyzeContextPropagation(t *testing.T) {
	results := analysistest.Run(t, analysistest.TestData(), statsAnalyzer, "stats")
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	got, ok := results[0].Result.(*contextpropagation.ContextPropagationInfo)
	if !ok {
		t.Fatalf("result is %T, want *ContextPropagationInfo", results[0].Result)
	}
	want := contextpropagation.ContextPropagationInfo{
		FunctionsWithContext:    3,
		FunctionsWithoutContext: 2,
	}
	if *got != want {
		t.Errorf("AnalyzeContextPropagation(stats) = %+v, want %+v", *got, want)
	}
}
