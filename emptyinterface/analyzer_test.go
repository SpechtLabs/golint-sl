package emptyinterface_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/spechtlabs/golint-sl/emptyinterface"
)

func TestEmptyInterfaceAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, emptyinterface.Analyzer, "a")
}
