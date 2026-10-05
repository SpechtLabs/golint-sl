package nopanic_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/spechtlabs/golint-sl/nopanic"
)

func TestNoPanicAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, nopanic.Analyzer, "a", "mainpkg")
}
