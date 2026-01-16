//go:build ignore
// +build ignore

// Package main provides a golangci-lint plugin for golint-sl analyzers.
//
// Build as a plugin:
//
//	go build -buildmode=plugin -o golint-sl.so ./plugin
//
// Then configure golangci-lint:
//
//	linters-settings:
//	  custom:
//	    golint-sl:
//	      path: ./golint-sl.so
//	      description: SpechtLabs code quality checks
//	      original-url: github.com/spechtlabs/golint-sl
//
// NOTE: This file is excluded from normal builds. Use -buildmode=plugin explicitly.
package main

import (
	"golang.org/x/tools/go/analysis"

	"github.com/spechtlabs/golint-sl/analyzers"
)

// AnalyzerPlugin exports the analyzers for golangci-lint plugin system.
var AnalyzerPlugin analyzerPlugin

type analyzerPlugin struct{}

// GetAnalyzers returns all golint-sl analyzers for the plugin system.
func (analyzerPlugin) GetAnalyzers() []*analysis.Analyzer {
	return analyzers.All()
}

// main is a no-op; this package is meant to be built with -buildmode=plugin.
func main() {}
