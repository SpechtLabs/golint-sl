package goroutineleak_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/spechtlabs/golint-sl/goroutineleak"
)

func TestGoroutineLeakAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, goroutineleak.Analyzer, "a")
}
