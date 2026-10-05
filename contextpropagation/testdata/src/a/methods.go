package a

import (
	"context"
	"time"
)

// Runner has an Exec method but no ExecContext to switch to.
type Runner struct{}

func (Runner) Exec() {}

// Querier is an interface with both variants.
type Querier interface {
	Query(args ...any)
	QueryContext(ctx context.Context, args ...any)
}

// Bad: each method call without a context is reported once
func (s *Store) WithoutContext(ctx context.Context) {
	helper(ctx)
	s.db.Query("SELECT 1")    // want `Query\(\) called without context as first argument; use QueryContext instead`
	s.db.QueryRow("SELECT 1") // want `QueryRow\(\) called without context as first argument; use QueryRowContext instead`
	s.db.Exec("DELETE")       // want `Exec\(\) called without context as first argument; use ExecContext instead`
	s.db.Prepare("SELECT 1")  // want `Prepare\(\) called without context as first argument; use PrepareContext instead`
	s.db.Begin()              // want `Begin\(\) called without context as first argument; use BeginTx instead`
}

// Bad: an interface with a context variant
func ThroughInterface(ctx context.Context, q Querier) {
	helper(ctx)
	q.Query("SELECT 1") // want `Query\(\) called without context as first argument; use QueryContext instead`
}

// Good: a derived context counts as a context whatever it is called
func Derived(ctx context.Context, db DB) {
	c, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	db.Query(c, "SELECT 1")
}

// Good: a copy of the context parameter that is passed on is propagation
func Copied(ctx context.Context, db DB) {
	c := ctx
	db.QueryContext(c, "x")
	db.QueryContext(c, "y")
}

// Good: a copy made with var works the same way
func CopiedVar(ctx context.Context, db DB) {
	var c = ctx
	db.QueryContext(c, "x")
	db.QueryContext(c, "y")
}

// Good: Runner has no ExecContext, so there is nothing better to call
func Run(ctx context.Context, r Runner) {
	r.Exec()
	_ = ctx.Err()
}

// Bad: a copy that is never passed on is still not propagation
func CopiedNotPassed(ctx context.Context) { // want `context parameter "ctx" is not passed to any sub-function calls`
	c := ctx
	_ = c
	helper(1)
	helper(2)
}
