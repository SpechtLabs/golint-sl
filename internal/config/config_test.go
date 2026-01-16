package config

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis"
)

func TestFilterAnalyzers(t *testing.T) {
	// Create mock analyzers
	mockAnalyzers := []*analysis.Analyzer{
		{Name: "analyzer1"},
		{Name: "analyzer2"},
		{Name: "analyzer3"},
	}

	tests := []struct {
		name   string
		config *Config
		want   []string
	}{
		{
			name:   "nil config enables all",
			config: nil,
			want:   []string{"analyzer1", "analyzer2", "analyzer3"},
		},
		{
			name: "default true enables all",
			config: &Config{
				Analyzers: map[string]bool{"default": true},
			},
			want: []string{"analyzer1", "analyzer2", "analyzer3"},
		},
		{
			name: "default false disables all",
			config: &Config{
				Analyzers: map[string]bool{"default": false},
			},
			want: []string{},
		},
		{
			name: "disable specific analyzer",
			config: &Config{
				Analyzers: map[string]bool{
					"default":   true,
					"analyzer2": false,
				},
			},
			want: []string{"analyzer1", "analyzer3"},
		},
		{
			name: "enable specific analyzers when default is false",
			config: &Config{
				Analyzers: map[string]bool{
					"default":   false,
					"analyzer1": true,
					"analyzer3": true,
				},
			},
			want: []string{"analyzer1", "analyzer3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.FilterAnalyzers(mockAnalyzers)
			if len(got) != len(tt.want) {
				t.Errorf("FilterAnalyzers() returned %d analyzers, want %d", len(got), len(tt.want))
				return
			}
			for i, a := range got {
				if a.Name != tt.want[i] {
					t.Errorf("FilterAnalyzers()[%d].Name = %q, want %q", i, a.Name, tt.want[i])
				}
			}
		})
	}
}

func TestIsEnabled(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		analyzer    string
		wantEnabled bool
	}{
		{
			name:        "nil config enables all",
			config:      nil,
			analyzer:    "any",
			wantEnabled: true,
		},
		{
			name: "explicitly enabled",
			config: &Config{
				Analyzers: map[string]bool{"myanalyzer": true},
			},
			analyzer:    "myanalyzer",
			wantEnabled: true,
		},
		{
			name: "explicitly disabled",
			config: &Config{
				Analyzers: map[string]bool{"myanalyzer": false},
			},
			analyzer:    "myanalyzer",
			wantEnabled: false,
		},
		{
			name: "uses default when not specified",
			config: &Config{
				Analyzers: map[string]bool{"default": false},
			},
			analyzer:    "other",
			wantEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.IsEnabled(tt.analyzer)
			if got != tt.wantEnabled {
				t.Errorf("IsEnabled(%q) = %v, want %v", tt.analyzer, got, tt.wantEnabled)
			}
		})
	}
}

func TestLoadFrom(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".golint-sl.yaml")

	configContent := `analyzers:
  default: true
  humaneerror: false
  todotracker: false
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := LoadFrom(configPath)
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}

	if cfg.Analyzers["default"] != true {
		t.Errorf("default = %v, want true", cfg.Analyzers["default"])
	}
	if cfg.Analyzers["humaneerror"] != false {
		t.Errorf("humaneerror = %v, want false", cfg.Analyzers["humaneerror"])
	}
	if cfg.Analyzers["todotracker"] != false {
		t.Errorf("todotracker = %v, want false", cfg.Analyzers["todotracker"])
	}
}
