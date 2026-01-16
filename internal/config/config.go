// Package config provides configuration file support for golint-sl.
package config

import (
	"os"
	"path/filepath"

	"golang.org/x/tools/go/analysis"
	"gopkg.in/yaml.v3"
)

// ConfigFileName is the default configuration file name.
const ConfigFileName = ".golint-sl.yaml"

// Config represents the golint-sl configuration.
type Config struct {
	// Analyzers configures which analyzers are enabled/disabled.
	// Use "default: false" to disable all by default, then enable specific ones.
	// Use "default: true" (or omit) to enable all by default, then disable specific ones.
	Analyzers map[string]bool `yaml:"analyzers"`
}

// Load attempts to load configuration from .golint-sl.yaml in the current
// directory or any parent directory up to the filesystem root.
func Load() (*Config, error) {
	path, err := findConfigFile()
	if err != nil {
		return nil, err
	}
	if path == "" {
		// No config file found, return default config
		return &Config{
			Analyzers: map[string]bool{"default": true},
		}, nil
	}

	return LoadFrom(path)
}

// LoadFrom loads configuration from the specified path.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Ensure Analyzers map exists
	if cfg.Analyzers == nil {
		cfg.Analyzers = map[string]bool{"default": true}
	}

	return &cfg, nil
}

// findConfigFile searches for .golint-sl.yaml starting from the current
// directory and walking up to parent directories.
func findConfigFile() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		configPath := filepath.Join(dir, ConfigFileName)
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return "", nil
		}
		dir = parent
	}
}

// FilterAnalyzers returns only the analyzers that are enabled according to the config.
func (c *Config) FilterAnalyzers(all []*analysis.Analyzer) []*analysis.Analyzer {
	if c == nil || c.Analyzers == nil {
		return all
	}

	// Check default setting
	defaultEnabled := true
	if val, ok := c.Analyzers["default"]; ok {
		defaultEnabled = val
	}

	var enabled []*analysis.Analyzer
	for _, a := range all {
		// Check if this specific analyzer has an override
		if val, ok := c.Analyzers[a.Name]; ok {
			if val {
				enabled = append(enabled, a)
			}
			// Explicitly disabled, skip
			continue
		}

		// Use default setting
		if defaultEnabled {
			enabled = append(enabled, a)
		}
	}

	return enabled
}

// IsEnabled checks if a specific analyzer is enabled.
func (c *Config) IsEnabled(name string) bool {
	if c == nil || c.Analyzers == nil {
		return true
	}

	// Check specific setting
	if val, ok := c.Analyzers[name]; ok {
		return val
	}

	// Check default
	if val, ok := c.Analyzers["default"]; ok {
		return val
	}

	return true
}
