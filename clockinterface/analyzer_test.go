package clockinterface_test

import (
	"reflect"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/spechtlabs/golint-sl/clockinterface"
)

func TestClockInterfaceAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, clockinterface.Analyzer,
		"a",
		"withclock",
		"ext",
		"tools/main",
		"example.com/cmd/tool",
		"example.com/ui/widgets",
		"example.com/domain",
		"example.com/main/store",
		"example.com/app",
	)
}

// patternAnalyzer exposes AnalyzeClockPattern as an analyzer result so it can
// run under analysistest.
var patternAnalyzer = &analysis.Analyzer{
	Name:       "clockpattern",
	Doc:        "test wrapper around clockinterface.AnalyzeClockPattern",
	Requires:   []*analysis.Analyzer{inspect.Analyzer},
	ResultType: reflect.TypeFor[*clockinterface.ClockPatternInfo](),
	Run: func(pass *analysis.Pass) (any, error) {
		return clockinterface.AnalyzeClockPattern(pass), nil
	},
}

func TestAnalyzeClockPattern(t *testing.T) {
	tests := []struct {
		pkg  string
		want clockinterface.ClockPatternInfo
	}{
		{
			pkg: "pattern",
			want: clockinterface.ClockPatternInfo{
				HasClockInterface:    true,
				HasRealClock:         true,
				HasMockClock:         true,
				DirectTimeNowCalls:   3,
				DirectTimeAfterCalls: 1,
			},
		},
		{
			// plain.Clock is a struct, so it is not reported as an interface.
			pkg: "plain",
			want: clockinterface.ClockPatternInfo{
				HasMockClock:       true,
				DirectTimeNowCalls: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.pkg, func(t *testing.T) {
			results := analysistest.Run(t, analysistest.TestData(), patternAnalyzer, tt.pkg)
			if len(results) != 1 {
				t.Fatalf("got %d results, want 1", len(results))
			}
			got, ok := results[0].Result.(*clockinterface.ClockPatternInfo)
			if !ok {
				t.Fatalf("result is %T, want *ClockPatternInfo", results[0].Result)
			}
			if *got != tt.want {
				t.Errorf("AnalyzeClockPattern(%s) = %+v, want %+v", tt.pkg, *got, tt.want)
			}
		})
	}
}
