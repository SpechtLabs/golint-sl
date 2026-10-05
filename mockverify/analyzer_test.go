package mockverify_test

import (
	"fmt"
	"slices"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
	"golang.org/x/tools/go/analysis/passes/inspect"

	"github.com/spechtlabs/golint-sl/mockverify"
)

func TestMockVerifyAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, mockverify.Analyzer, "a", "b/mock")
}

// infoAnalyzer reports the result of AnalyzeMocks as one diagnostic on the
// package clause, with every list sorted so the output is deterministic.
var infoAnalyzer = &analysis.Analyzer{
	Name:     "mockinfo",
	Doc:      "reports mockverify.AnalyzeMocks results",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run: func(pass *analysis.Pass) (any, error) {
		info := mockverify.AnalyzeMocks(pass)
		for _, list := range [][]string{info.Mocks, info.VerifiedMocks, info.UnverifiedMocks} {
			slices.Sort(list)
		}
		pass.Reportf(pass.Files[0].Package, "%s", fmt.Sprintf("mocks=%v verified=%v unverified=%v",
			info.Mocks, info.VerifiedMocks, info.UnverifiedMocks))
		return nil, nil
	},
}

func TestAnalyzeMocks(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, infoAnalyzer, "info")
}
