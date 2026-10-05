package taint

import (
	"database/sql"
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

type Runner struct{}

func (Runner) Exec(cmd string) {}

func generic[T any](v T) {}

func Flow(db *sql.DB, r Runner, tainted string) {
	_ = slog.String("k", tainted)     // want `logging sink reached from Flow.tainted`
	_ = exec.Command(tainted)         // want `command_execution sink reached from Flow.tainted`
	r.Exec(tainted)                   // want `command_execution sink reached from Flow.tainted`
	_, _ = db.Query(tainted)          // want `sql_query sink reached from Flow.tainted`
	_ = os.WriteFile(tainted, nil, 0) // want `file_write sink reached from Flow.tainted`
	upper := strings.ToUpper(tainted)
	_, _ = os.Create(upper) // want `file_write sink reached from Flow.tainted`
	_ = strings.TrimSpace(tainted)
	generic(tainted)
	_ = os.Getenv(tainted)
}

func Clean(db *sql.DB, input string) {
	_, _ = db.Query(input)
}
