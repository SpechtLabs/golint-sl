package closurecomplexity_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/spechtlabs/golint-sl/closurecomplexity"
)

func TestClosureComplexityAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, closurecomplexity.Analyzer, "a", "nofuncs")
}
