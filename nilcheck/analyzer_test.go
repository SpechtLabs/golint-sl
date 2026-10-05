package nilcheck_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/spechtlabs/golint-sl/nilcheck"
)

func TestNilCheckAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, nilcheck.Analyzer, "a", "b/mocks")
}
