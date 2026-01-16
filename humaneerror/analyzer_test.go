package humaneerror_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/spechtlabs/golint-sl/humaneerror"
)

func TestHumaneErrorAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, humaneerror.Analyzer, "a")
}
