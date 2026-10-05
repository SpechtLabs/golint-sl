package contextlogger_test

import (
	"reflect"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/spechtlabs/golint-sl/contextlogger"
)

func TestContextLoggerAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, contextlogger.Analyzer, "a")
}

// infoAnalyzer exposes AnalyzeContextLogger as an analyzer result so it can
// run under analysistest.
var infoAnalyzer = &analysis.Analyzer{
	Name:       "contextloggerinfo",
	Doc:        "test wrapper around contextlogger.AnalyzeContextLogger",
	Requires:   []*analysis.Analyzer{inspect.Analyzer},
	ResultType: reflect.TypeFor[*contextlogger.ContextLoggerInfo](),
	Run: func(pass *analysis.Pass) (any, error) {
		return contextlogger.AnalyzeContextLogger(pass), nil
	},
}

func TestAnalyzeContextLogger(t *testing.T) {
	tests := []struct {
		pkg  string
		want contextlogger.ContextLoggerInfo
	}{
		{
			pkg: "info",
			want: contextlogger.ContextLoggerInfo{
				HasFromContext: true,
				HasIntoContext: true,
				// FromContext(ctx) twice in use, once inside FromContext itself.
				ContextLoggerCalls: 3,
				// log.Info, log.Error and stdlog.Printf (counted once despite
				// matching both "log.Print" and "log.Printf").
				GlobalLoggerCalls: 3,
			},
		},
		{
			pkg: "github.com/sirupsen/logrus",
		},
	}

	for _, tt := range tests {
		t.Run(tt.pkg, func(t *testing.T) {
			results := analysistest.Run(t, analysistest.TestData(), infoAnalyzer, tt.pkg)
			if len(results) != 1 {
				t.Fatalf("got %d results, want 1", len(results))
			}
			got, ok := results[0].Result.(*contextlogger.ContextLoggerInfo)
			if !ok {
				t.Fatalf("result is %T, want *ContextLoggerInfo", results[0].Result)
			}
			if *got != tt.want {
				t.Errorf("AnalyzeContextLogger(%s) = %+v, want %+v", tt.pkg, *got, tt.want)
			}
		})
	}
}
