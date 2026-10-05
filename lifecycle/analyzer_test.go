package lifecycle_test

import (
	"fmt"
	"slices"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/spechtlabs/golint-sl/lifecycle"
)

func TestLifecycleAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, lifecycle.Analyzer, "a")
}

// infoAnalyzer reports the result of AnalyzeLifecycle as one diagnostic on
// the package clause, with every list sorted so the output is deterministic.
var infoAnalyzer = &analysis.Analyzer{
	Name:     "lifecycleinfo",
	Doc:      "reports lifecycle.AnalyzeLifecycle results",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run: func(pass *analysis.Pass) (any, error) {
		info := lifecycle.AnalyzeLifecycle(pass)
		for _, list := range [][]string{info.TypesWithRun, info.TypesWithStop, info.TypesMissingStop, info.TypesWithBothRun} {
			slices.Sort(list)
		}
		pass.Reportf(pass.Files[0].Package, "%s", fmt.Sprintf("run=%v stop=%v missing=%v both=%v",
			info.TypesWithRun, info.TypesWithStop, info.TypesMissingStop, info.TypesWithBothRun))
		return nil, nil
	},
}

func TestAnalyzeLifecycle(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, infoAnalyzer, "info")
}
