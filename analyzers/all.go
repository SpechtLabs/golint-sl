// Package analyzers provides a registry of all golint-sl analyzers.
//
// This package exports all analyzers in a single slice for convenient use
// with multichecker and plugin systems.
package analyzers

import (
	"golang.org/x/tools/go/analysis"

	"github.com/spechtlabs/golint-sl/clockinterface"
	"github.com/spechtlabs/golint-sl/closurecomplexity"
	"github.com/spechtlabs/golint-sl/contextfirst"
	"github.com/spechtlabs/golint-sl/contextlogger"
	"github.com/spechtlabs/golint-sl/contextpropagation"
	"github.com/spechtlabs/golint-sl/dataflow"
	"github.com/spechtlabs/golint-sl/emptyinterface"
	"github.com/spechtlabs/golint-sl/errorwrap"
	"github.com/spechtlabs/golint-sl/exporteddoc"
	"github.com/spechtlabs/golint-sl/functionsize"
	"github.com/spechtlabs/golint-sl/goroutineleak"
	"github.com/spechtlabs/golint-sl/hardcodedcreds"
	"github.com/spechtlabs/golint-sl/httpclient"
	"github.com/spechtlabs/golint-sl/humaneerror"
	"github.com/spechtlabs/golint-sl/interfaceconsistency"
	"github.com/spechtlabs/golint-sl/lifecycle"
	"github.com/spechtlabs/golint-sl/mockverify"
	"github.com/spechtlabs/golint-sl/nestingdepth"
	"github.com/spechtlabs/golint-sl/nilcheck"
	"github.com/spechtlabs/golint-sl/nopanic"
	"github.com/spechtlabs/golint-sl/optionspattern"
	"github.com/spechtlabs/golint-sl/pkgnaming"
	"github.com/spechtlabs/golint-sl/reconciler"
	"github.com/spechtlabs/golint-sl/resourceclose"
	"github.com/spechtlabs/golint-sl/returninterface"
	"github.com/spechtlabs/golint-sl/sentinelerrors"
	"github.com/spechtlabs/golint-sl/sideeffects"
	"github.com/spechtlabs/golint-sl/statusupdate"
	"github.com/spechtlabs/golint-sl/syncaccess"
	"github.com/spechtlabs/golint-sl/todotracker"
	"github.com/spechtlabs/golint-sl/wideevents"
)

// All returns all available analyzers.
// Analyzers are grouped by category for clarity.
func All() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		// Error Handling
		humaneerror.Analyzer,
		errorwrap.Analyzer,
		sentinelerrors.Analyzer,

		// Observability
		wideevents.Analyzer,
		contextlogger.Analyzer,
		contextpropagation.Analyzer,

		// Kubernetes
		reconciler.Analyzer,
		statusupdate.Analyzer,
		sideeffects.Analyzer,

		// Testability
		clockinterface.Analyzer,
		interfaceconsistency.Analyzer,
		mockverify.Analyzer,
		optionspattern.Analyzer,

		// Resources
		resourceclose.Analyzer,
		httpclient.Analyzer,

		// Safety
		goroutineleak.Analyzer,
		nilcheck.Analyzer,
		nopanic.Analyzer,
		nestingdepth.Analyzer,
		syncaccess.Analyzer,

		// Clean Code
		closurecomplexity.Analyzer,
		emptyinterface.Analyzer,
		returninterface.Analyzer,

		// Architecture
		contextfirst.Analyzer,
		pkgnaming.Analyzer,
		functionsize.Analyzer,
		exporteddoc.Analyzer,
		todotracker.Analyzer,
		hardcodedcreds.Analyzer,
		lifecycle.Analyzer,
		dataflow.Analyzer,
	}
}

// ErrorHandling returns analyzers focused on error handling patterns.
func ErrorHandling() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		humaneerror.Analyzer,
		errorwrap.Analyzer,
		sentinelerrors.Analyzer,
	}
}

// Observability returns analyzers focused on logging and observability.
func Observability() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		wideevents.Analyzer,
		contextlogger.Analyzer,
		contextpropagation.Analyzer,
	}
}

// Kubernetes returns analyzers focused on Kubernetes patterns.
func Kubernetes() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		reconciler.Analyzer,
		statusupdate.Analyzer,
		sideeffects.Analyzer,
	}
}

// Testability returns analyzers focused on testable code patterns.
func Testability() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		clockinterface.Analyzer,
		interfaceconsistency.Analyzer,
		mockverify.Analyzer,
		optionspattern.Analyzer,
	}
}

// Resources returns analyzers focused on resource management.
func Resources() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		resourceclose.Analyzer,
		httpclient.Analyzer,
	}
}

// Safety returns analyzers focused on code safety.
func Safety() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		goroutineleak.Analyzer,
		nilcheck.Analyzer,
		nopanic.Analyzer,
		nestingdepth.Analyzer,
		syncaccess.Analyzer,
	}
}

// CleanCode returns analyzers focused on clean code patterns.
func CleanCode() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		closurecomplexity.Analyzer,
		emptyinterface.Analyzer,
		returninterface.Analyzer,
	}
}

// Architecture returns analyzers focused on architectural patterns.
func Architecture() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		contextfirst.Analyzer,
		pkgnaming.Analyzer,
		functionsize.Analyzer,
		exporteddoc.Analyzer,
		todotracker.Analyzer,
		hardcodedcreds.Analyzer,
		lifecycle.Analyzer,
		dataflow.Analyzer,
	}
}
