package pkgnaming_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/spechtlabs/golint-sl/pkgnaming"
)

func TestPkgNamingAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, pkgnaming.Analyzer,
		"user", "http", "util", "helpers", "my_pkg", "myPkg",
		"handlers", "users", "my_utils", "status", "bus", "mock")
}
