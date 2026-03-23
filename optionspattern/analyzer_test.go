package optionspattern_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/spechtlabs/golint-sl/optionspattern"
)

func TestOptionsPatternAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, optionspattern.Analyzer, "a")
}
