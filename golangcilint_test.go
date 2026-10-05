package golintsl_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/golangci/plugin-module-register/register"

	golintsl "github.com/spechtlabs/golint-sl"
	"github.com/spechtlabs/golint-sl/analyzers"
)

func analyzerNames(t *testing.T, p register.LinterPlugin) []string {
	t.Helper()
	as, err := p.BuildAnalyzers()
	if err != nil {
		t.Fatalf("BuildAnalyzers() error = %v", err)
	}
	names := make([]string, 0, len(as))
	for _, a := range as {
		names = append(names, a.Name)
	}
	return names
}

func allNamesExcept(skip ...string) []string {
	var names []string
	for _, a := range analyzers.All() {
		if !slices.Contains(skip, a.Name) {
			names = append(names, a.Name)
		}
	}
	return names
}

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		conf any
		want []string
	}{
		{
			name: "no settings enables every analyzer",
			conf: nil,
			want: allNamesExcept(),
		},
		{
			name: "empty settings enable every analyzer",
			conf: map[string]any{},
			want: allNamesExcept(),
		},
		{
			name: "empty disabled list enables every analyzer",
			conf: map[string]any{"disabled-analyzers": []string{}},
			want: allNamesExcept(),
		},
		{
			name: "disabled analyzers are filtered out",
			conf: map[string]any{"disabled-analyzers": []string{"wideevents", "nilcheck"}},
			want: allNamesExcept("wideevents", "nilcheck"),
		},
		{
			name: "typed settings are accepted",
			conf: golintsl.Settings{DisabledAnalyzers: []string{"todotracker"}},
			want: allNamesExcept("todotracker"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := golintsl.New(tt.conf)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			if got := analyzerNames(t, p); !slices.Equal(got, tt.want) {
				t.Errorf("BuildAnalyzers() names = %v, want %v", got, tt.want)
			}
			if got := p.GetLoadMode(); got != register.LoadModeTypesInfo {
				t.Errorf("GetLoadMode() = %q, want %q", got, register.LoadModeTypesInfo)
			}
		})
	}
}

func TestNewRejectsBadSettings(t *testing.T) {
	tests := []struct {
		name    string
		conf    any
		wantErr string
	}{
		{
			name:    "a misspelled key",
			conf:    map[string]any{"disabled_analyzers": []string{"wideevents"}},
			wantErr: `unknown field "disabled_analyzers"`,
		},
		{
			name:    "a list given as a string",
			conf:    map[string]any{"disabled-analyzers": "wideevents"},
			wantErr: "golint-sl settings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := golintsl.New(tt.conf)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("New() error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestBuildAnalyzersRejectsUnknownNames(t *testing.T) {
	p, err := golintsl.New(map[string]any{"disabled-analyzers": []string{"nilcheck", "doesnotexist"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.BuildAnalyzers(); err == nil || !strings.Contains(err.Error(), `"doesnotexist"`) {
		t.Fatalf("BuildAnalyzers() error = %v, want the unknown name reported", err)
	}
}

func TestPluginRegistered(t *testing.T) {
	newPlugin, err := register.GetPlugin("golint-sl")
	if err != nil {
		t.Fatalf("GetPlugin(golint-sl) error = %v", err)
	}
	p, err := newPlugin(nil)
	if err != nil {
		t.Fatalf("plugin constructor error = %v", err)
	}
	if got, want := len(analyzerNames(t, p)), len(analyzers.All()); got != want {
		t.Errorf("registered plugin builds %d analyzers, want %d", got, want)
	}
}
