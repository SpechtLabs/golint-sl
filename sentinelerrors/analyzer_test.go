package sentinelerrors_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/spechtlabs/golint-sl/sentinelerrors"
)

func TestSentinelErrorsAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, sentinelerrors.Analyzer, "a", "mainpkg")
}
