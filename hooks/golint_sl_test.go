// Package hooks_test drives hooks/golint-sl.sh, the script behind the
// pre-commit hooks, against throwaway repositories. A fake golangci-lint on
// PATH writes a fake custom-gcl that records where it ran and with which
// arguments, so the tests see which modules and packages the hook lints
// without building golangci-lint.
package hooks_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// fakeGolangciLint is the golangci-lint the tests put on PATH. `custom`
// writes ./custom-gcl, which appends "<module dir>|<args>" to $HOOK_LOG and
// fails in the module named by $FAIL_MODULE.
const fakeGolangciLint = `#!/usr/bin/env bash
set -eu
[ "$1" = custom ] || exit 2
cat > custom-gcl <<'GCL'
#!/usr/bin/env bash
dir="$(git rev-parse --show-prefix)"
dir="${dir%/}"
dir="${dir:-.}"
echo "${dir}|$*" >> "$HOOK_LOG"
[ "${dir}" != "${FAIL_MODULE:-}" ]
GCL
chmod +x custom-gcl
`

func TestHook(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("the hook is a bash script")
	}

	tests := []struct {
		name       string
		files      []string
		args       []string
		noLinter   bool
		failModule string
		wantRuns   []string
		wantErr    bool
		wantStderr string
	}{
		{
			name:     "a go.mod at the root lints the root module",
			files:    []string{".custom-gcl.yml", "go.mod", "main.go", "pkg/p.go"},
			wantRuns: []string{".|run ./..."},
		},
		{
			name: "modules in subdirectories only are linted one by one",
			files: []string{
				".custom-gcl.yml",
				"otelzap/go.mod", "otelzap/zap.go",
				"tools/gen/go.mod", "tools/gen/main.go",
			},
			wantRuns: []string{"otelzap|run ./...", "tools/gen|run ./..."},
		},
		{
			name: "a root module and nested modules are each linted",
			files: []string{
				".custom-gcl.yml", "go.mod", "main.go",
				"example/go.mod", "example/main.go",
			},
			wantRuns: []string{".|run ./...", "example|run ./..."},
		},
		{
			name: "modules under testdata, vendor, and dot or underscore directories are skipped",
			files: []string{
				".custom-gcl.yml", "go.mod",
				"pkg/testdata/src/go.mod", "vendor/x/go.mod", ".tools/go.mod", "_scratch/go.mod",
			},
			wantRuns: []string{".|run ./..."},
		},
		{
			name: "--packages lints the changed files' packages, grouped by module",
			files: []string{
				".custom-gcl.yml",
				"otelzap/go.mod", "otelzap/zap.go", "otelzap/internal/field/field.go",
				"tools/gen/go.mod", "tools/gen/main.go",
			},
			args: []string{
				"--packages",
				"otelzap/zap.go", "otelzap/internal/field/field.go", "otelzap/zap.go",
				"tools/gen/main.go",
			},
			wantRuns: []string{"otelzap|run . ./internal/field", "tools/gen|run ."},
		},
		{
			name: "--packages with a root module keeps nested modules apart",
			files: []string{
				".custom-gcl.yml", "go.mod", "main.go", "pkg/p.go",
				"example/go.mod", "example/main.go",
			},
			args:     []string{"--packages", "main.go", "pkg/p.go", "example/main.go"},
			wantRuns: []string{".|run . ./pkg", "example|run ."},
		},
		{
			name:     "--packages skips files outside every module",
			files:    []string{".custom-gcl.yml", "a/go.mod", "a/a.go", "scripts/gen.go"},
			args:     []string{"--packages", "scripts/gen.go", "a/a.go"},
			wantRuns: []string{"a|run ."},
		},
		{
			name: "a failing module fails the hook after every module ran",
			files: []string{
				".custom-gcl.yml", "a/go.mod", "a/a.go", "b/go.mod", "b/b.go",
			},
			failModule: "a",
			wantRuns:   []string{"a|run ./...", "b|run ./..."},
			wantErr:    true,
		},
		{
			name:       "without .custom-gcl.yml the hook explains what's missing",
			files:      []string{"go.mod", "main.go"},
			wantErr:    true,
			wantStderr: "no .custom-gcl.yml at the repository root",
		},
		{
			name:       "without golangci-lint or mise the hook says so",
			files:      []string{".custom-gcl.yml", "go.mod", "main.go"},
			noLinter:   true,
			wantErr:    true,
			wantStderr: "golangci-lint is neither on PATH nor available through mise",
		},
	}

	script, err := filepath.Abs("golint-sl.sh")
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := newRepo(t, tt.files)
			log := filepath.Join(t.TempDir(), "runs.log")

			cmd := exec.Command(script, tt.args...)
			cmd.Dir = repo
			cmd.Env = []string{
				"PATH=" + toolPath(t, !tt.noLinter),
				"HOME=" + t.TempDir(),
				"HOOK_LOG=" + log,
				"FAIL_MODULE=" + tt.failModule,
			}
			var stderr strings.Builder
			cmd.Stderr = &stderr
			err := cmd.Run()

			if gotErr := err != nil; gotErr != tt.wantErr {
				t.Fatalf("hook error = %v, want error %v; stderr:\n%s", err, tt.wantErr, stderr.String())
			}
			if !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("stderr = %q, want it to contain %q", stderr.String(), tt.wantStderr)
			}
			if got := readRuns(t, log); !slices.Equal(got, tt.wantRuns) {
				t.Errorf("custom-gcl runs = %q, want %q", got, tt.wantRuns)
			}
		})
	}
}

// newRepo creates a Git repository holding the given files, all empty but the
// go.mod files, and returns its path.
func newRepo(t *testing.T, files []string) string {
	t.Helper()
	repo := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", repo).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, name := range files {
		path := filepath.Join(repo, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		content := ""
		if filepath.Base(name) == "go.mod" {
			content = "module example.com/m\n\ngo 1.27\n"
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return repo
}

// toolPath returns a PATH with git and the shell utilities the hook needs,
// and with the fake golangci-lint when withLinter is set. git is linked into
// a directory of its own, so whatever else sits next to it (a real
// golangci-lint, or mise) stays out of reach.
func toolPath(t *testing.T, withLinter bool) string {
	t.Helper()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	if err := os.Symlink(gitPath, filepath.Join(bin, "git")); err != nil {
		t.Fatal(err)
	}
	dirs := []string{bin, "/usr/bin", "/bin"}
	if withLinter {
		if err := os.WriteFile(filepath.Join(bin, "golangci-lint"), []byte(fakeGolangciLint), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	return strings.Join(dirs, string(os.PathListSeparator))
}

// readRuns returns the custom-gcl runs the fake recorded, in order.
func readRuns(t *testing.T, log string) []string {
	t.Helper()
	data, err := os.ReadFile(log)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSpace(string(data)), "\n")
}
