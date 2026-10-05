package wideevents_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/spechtlabs/golint-sl/wideevents"
)

func TestWideEventsAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, wideevents.Analyzer, "a")
}

func TestWideEventsAnalyzerCLIPackage(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, wideevents.Analyzer, "example.com/tool/cmd")
}
