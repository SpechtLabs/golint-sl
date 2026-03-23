package interfaceconsistency_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/spechtlabs/golint-sl/interfaceconsistency"
)

func TestInterfaceConsistencyAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, interfaceconsistency.Analyzer, "a")
}

func TestInterfaceConsistencyMainPackage(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, interfaceconsistency.Analyzer, "mainpkg")
}
