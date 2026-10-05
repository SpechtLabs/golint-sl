package statusupdate_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/spechtlabs/golint-sl/statusupdate"
)

func TestStatusUpdateAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, statusupdate.Analyzer, "a")
}
