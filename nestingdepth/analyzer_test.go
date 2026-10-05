package nestingdepth_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/spechtlabs/golint-sl/nestingdepth"
)

func TestNestingDepthAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, nestingdepth.Analyzer, "a")
}
