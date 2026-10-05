package hardcodedcreds_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/spechtlabs/golint-sl/hardcodedcreds"
)

func TestHardcodedCredsAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, hardcodedcreds.Analyzer, "a")
}
