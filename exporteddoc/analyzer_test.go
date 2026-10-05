package exporteddoc_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/spechtlabs/golint-sl/exporteddoc"
)

func TestExportedDocAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, exporteddoc.Analyzer, "a")
}
