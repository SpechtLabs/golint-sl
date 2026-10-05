package dataflow_test

import (
	"fmt"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
	"golang.org/x/tools/go/analysis/passes/buildssa"

	"github.com/spechtlabs/golint-sl/dataflow"
)

func TestDataflowAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, dataflow.Analyzer, "a")
}

// taintAnalyzer drives the exported TaintAnalysis API: every parameter named
// "tainted" is a source, and each distinct sink it reaches is reported once.
var taintAnalyzer = &analysis.Analyzer{
	Name:     "tainttest",
	Doc:      "test wrapper around dataflow.TaintAnalysis",
	Requires: []*analysis.Analyzer{buildssa.Analyzer},
	Run: func(pass *analysis.Pass) (any, error) {
		ssaInfo := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)
		ta := dataflow.NewTaintAnalysis()
		for _, fn := range ssaInfo.SrcFuncs {
			// Functions have no referrers; marking one exercises the skip.
			ta.MarkSource(fn, "function "+fn.Name())
			for _, p := range fn.Params {
				if p.Name() == "tainted" {
					ta.MarkSource(p, fn.Name()+"."+p.Name())
				}
			}
		}
		ta.Propagate()

		seen := make(map[string]bool)
		for _, sink := range ta.Sinks {
			key := fmt.Sprintf("%d %s %s", sink.Call.Pos(), sink.SinkType, sink.Source)
			if seen[key] {
				continue
			}
			seen[key] = true
			pass.Reportf(sink.Call.Pos(), "%s sink reached from %s", sink.SinkType, sink.Source)
		}
		return nil, nil
	},
}

func TestTaintAnalysis(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), taintAnalyzer, "taint")
}
