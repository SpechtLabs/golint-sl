package analyzers_test

import (
	"testing"

	"golang.org/x/tools/go/analysis"

	"github.com/spechtlabs/golint-sl/analyzers"
)

func TestAllHasEveryAnalyzerOnce(t *testing.T) {
	all := analyzers.All()
	if len(all) == 0 {
		t.Fatal("All() returned no analyzers")
	}

	byName := make(map[string]*analysis.Analyzer, len(all))
	seen := make(map[*analysis.Analyzer]bool, len(all))
	for i, a := range all {
		if a == nil {
			t.Fatalf("All()[%d] is nil", i)
		}
		if seen[a] {
			t.Errorf("analyzer %q appears more than once", a.Name)
		}
		seen[a] = true
		if a.Name == "" {
			t.Errorf("All()[%d] has an empty name", i)
		}
		if _, dup := byName[a.Name]; dup {
			t.Errorf("analyzer name %q is not unique", a.Name)
		}
		byName[a.Name] = a
		if err := analysis.Validate([]*analysis.Analyzer{a}); err != nil {
			t.Errorf("analyzer %q is invalid: %v", a.Name, err)
		}
	}
}

func TestCategoriesPartitionAll(t *testing.T) {
	categories := []struct {
		name string
		fn   func() []*analysis.Analyzer
	}{
		{name: "ErrorHandling", fn: analyzers.ErrorHandling},
		{name: "Observability", fn: analyzers.Observability},
		{name: "Kubernetes", fn: analyzers.Kubernetes},
		{name: "Testability", fn: analyzers.Testability},
		{name: "Resources", fn: analyzers.Resources},
		{name: "Safety", fn: analyzers.Safety},
		{name: "CleanCode", fn: analyzers.CleanCode},
		{name: "Architecture", fn: analyzers.Architecture},
	}

	inAll := make(map[*analysis.Analyzer]bool)
	for _, a := range analyzers.All() {
		inAll[a] = true
	}

	category := make(map[*analysis.Analyzer]string)
	for _, c := range categories {
		t.Run(c.name, func(t *testing.T) {
			as := c.fn()
			if len(as) == 0 {
				t.Fatalf("%s() returned no analyzers", c.name)
			}
			for _, a := range as {
				if !inAll[a] {
					t.Errorf("%s() includes %q, which is missing from All()", c.name, a.Name)
				}
				if prev, ok := category[a]; ok {
					t.Errorf("%q is in both %s() and %s()", a.Name, prev, c.name)
				}
				category[a] = c.name
			}
		})
	}

	for a := range inAll {
		if _, ok := category[a]; !ok {
			t.Errorf("%q is in All() but in no category", a.Name)
		}
	}
}
