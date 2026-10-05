// Package golintsl provides golangci-lint v2 module plugin integration.
//
// This file registers golint-sl as a module plugin for golangci-lint v2.
// To use golint-sl with golangci-lint, you need to build a custom binary:
//
//  1. Create a .custom-gcl.yml file referencing this module
//  2. Run: golangci-lint custom
//  3. Use the generated ./custom-gcl binary
//
// See https://golangci-lint.run/plugins/module-plugins/ for more details.
package golintsl

import (
	"fmt"
	"slices"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"

	"github.com/spechtlabs/golint-sl/analyzers"
)

//nolint:gochecknoinits // Required for golangci-lint module plugin registration
func init() {
	register.Plugin("golint-sl", New)
}

// Settings allows configuring which analyzers to enable/disable.
type Settings struct {
	// DisabledAnalyzers is a list of analyzer names to disable.
	DisabledAnalyzers []string `json:"disabled-analyzers"`
}

type plugin struct {
	settings Settings
}

// New creates a new golint-sl plugin instance. Settings it can't decode, such
// as a misspelled key, are an error rather than silently ignored: otherwise
// a typo in disabled-analyzers would run every analyzer it meant to disable.
func New(conf any) (register.LinterPlugin, error) {
	s, err := register.DecodeSettings[Settings](conf)
	if err != nil {
		return nil, fmt.Errorf("golint-sl settings: %w (the only setting is disabled-analyzers, a list of analyzer names)", err)
	}
	return &plugin{settings: s}, nil
}

// BuildAnalyzers returns the list of analyzers to run.
func (p *plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	all := analyzers.All()
	if len(p.settings.DisabledAnalyzers) == 0 {
		return all, nil
	}

	// Filter out disabled analyzers. A name that matches no analyzer is most
	// likely a typo, which would leave that analyzer running.
	disabled := make(map[string]bool)
	for _, name := range p.settings.DisabledAnalyzers {
		if !slices.ContainsFunc(all, func(a *analysis.Analyzer) bool { return a.Name == name }) {
			return nil, fmt.Errorf("golint-sl settings: disabled-analyzers names %q, which is not a golint-sl analyzer", name)
		}
		disabled[name] = true
	}

	var result []*analysis.Analyzer
	for _, a := range all {
		if !disabled[a.Name] {
			result = append(result, a)
		}
	}
	return result, nil
}

// GetLoadMode returns the load mode required by the analyzers.
// Several golint-sl analyzers use pass.TypesInfo, so we need TypesInfo mode.
func (p *plugin) GetLoadMode() string {
	return register.LoadModeTypesInfo
}
