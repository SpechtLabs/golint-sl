package config

import (
	"os"
	"path/filepath"
	"reflect"
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
		{
			name:        "nil analyzers map enables all",
			config:      &Config{},
			analyzer:    "any",
			wantEnabled: true,
		},
		{
			name: "enabled when neither the analyzer nor default is set",
			config: &Config{
				Analyzers: map[string]bool{"someother": false},
			},
			analyzer:    "any",
			wantEnabled: true,
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
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
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

func TestLoadFromCases(t *testing.T) {
	tests := []struct {
		name    string
		content *string // nil: the file does not exist
		want    map[string]bool
		wantErr bool
	}{
		{
			name:    "missing file",
			content: nil,
			wantErr: true,
		},
		{
			name:    "invalid yaml",
			content: new("analyzers: [unclosed"),
			wantErr: true,
		},
		{
			name:    "wrong type for analyzers",
			content: new("analyzers: yes\n"),
			wantErr: true,
		},
		{
			name:    "empty file yields the default config",
			content: new(""),
			want:    map[string]bool{"default": true},
		},
		{
			name:    "no analyzers key yields the default config",
			content: new("something-else: 1\n"),
			want:    map[string]bool{"default": true},
		},
		{
			name:    "analyzers are read as given",
			content: new("analyzers:\n  default: false\n  nilcheck: true\n"),
			want:    map[string]bool{"default": false, "nilcheck": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), ConfigFileName)
			if tt.content != nil {
				if err := os.WriteFile(path, []byte(*tt.content), 0o600); err != nil {
					t.Fatalf("write config: %v", err)
				}
			}

			cfg, err := LoadFrom(path)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("LoadFrom() = %+v, want an error", cfg)
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadFrom() error = %v", err)
			}
			if !reflect.DeepEqual(cfg.Analyzers, tt.want) {
				t.Errorf("Analyzers = %v, want %v", cfg.Analyzers, tt.want)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name string
		// configDir is where the config file is written, relative to the
		// temp root; empty means no config file.
		configDir string
		content   string
		// workDir is the working directory, relative to the temp root.
		workDir string
		want    map[string]bool
		wantErr bool
	}{
		{
			name:    "no config file anywhere yields the default config",
			workDir: "project/sub",
			want:    map[string]bool{"default": true},
		},
		{
			name:      "config file in the working directory",
			configDir: "project",
			content:   "analyzers:\n  nilcheck: false\n",
			workDir:   "project",
			want:      map[string]bool{"nilcheck": false},
		},
		{
			name:      "config file in a parent directory",
			configDir: "project",
			content:   "analyzers:\n  default: false\n",
			workDir:   "project/sub/deeper",
			want:      map[string]bool{"default": false},
		},
		{
			name:      "nearest config file wins",
			configDir: "project/sub",
			content:   "analyzers:\n  todotracker: false\n",
			workDir:   "project/sub",
			want:      map[string]bool{"todotracker": false},
		},
		{
			name:      "invalid config file is an error",
			configDir: "project",
			content:   "analyzers: [unclosed",
			workDir:   "project/sub",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			workDir := filepath.Join(root, tt.workDir)
			if err := os.MkdirAll(workDir, 0o750); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			if tt.configDir != "" {
				path := filepath.Join(root, tt.configDir, ConfigFileName)
				if err := os.WriteFile(path, []byte(tt.content), 0o600); err != nil {
					t.Fatalf("write config: %v", err)
				}
			}
			t.Chdir(workDir)

			cfg, err := Load()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Load() = %+v, want an error", cfg)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if !reflect.DeepEqual(cfg.Analyzers, tt.want) {
				t.Errorf("Analyzers = %v, want %v", cfg.Analyzers, tt.want)
			}
		})
	}
}
