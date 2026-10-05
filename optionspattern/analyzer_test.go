package optionspattern_test

import (
	"reflect"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/spechtlabs/golint-sl/optionspattern"
)

func TestOptionsPatternAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, optionspattern.Analyzer, "a")
}

func TestAnalyzeOptionPatterns(t *testing.T) {
	collector := &analysis.Analyzer{
		Name:     "collectoptions",
		Doc:      "collects option pattern information for tests",
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Run: func(pass *analysis.Pass) (any, error) {
			return optionspattern.AnalyzeOptionPatterns(pass), nil
		},
		ResultType: reflect.TypeFor[*optionspattern.OptionPatternInfo](),
	}

	results := analysistest.Run(t, analysistest.TestData(), collector, "collect")
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	got, ok := results[0].Result.(*optionspattern.OptionPatternInfo)
	if !ok {
		t.Fatalf("result has type %T, want *OptionPatternInfo", results[0].Result)
	}

	want := &optionspattern.OptionPatternInfo{
		OptionTypes:     []string{"ClientOption"},
		OptionFunctions: []string{"WithRetry"},
		Constructors:    []string{"NewClient"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("AnalyzeOptionPatterns() = %+v, want %+v", got, want)
	}
}
