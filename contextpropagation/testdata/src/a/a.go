package a

import (
	"context"
	"net/http"
	"net/url"
	"os/exec"
	"time"

	"google.golang.org/grpc"
)

// DB mimics database/sql: methods with Context variants.
type DB struct{}

func (DB) Query(args ...any)    {}
func (DB) QueryRow(args ...any) {}
func (DB) Exec(args ...any)     {}
func (DB) Prepare(args ...any)  {}
func (DB) Begin(args ...any)    {}

func (DB) QueryContext(ctx context.Context, args ...any)    { _ = ctx }
func (DB) QueryRowContext(ctx context.Context, args ...any) { _ = ctx }
func (DB) ExecContext(ctx context.Context, args ...any)     { _ = ctx }
func (DB) PrepareContext(ctx context.Context, args ...any)  { _ = ctx }
func (DB) BeginTx(ctx context.Context, args ...any)         { _ = ctx }

type Store struct {
	db  DB
	dbs []DB
	ctx context.Context
	sub struct{ inner struct{ db DB } }
}

func (s *Store) getDB() DB { return s.db }

func helper(args ...any) {}

// Good: context passed to a sub-call
func Passed(ctx context.Context, id string) {
	helper(ctx, id)
}

// Good: context passed inside a nested expression
func PassedNested(ctx context.Context) {
	helper(struct{ c context.Context }{c: ctx})
}

// Good: context methods count as use
func Waits(ctx context.Context, ch chan int) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ch:
	}
	helper(1)
	helper(2)
	return nil
}

// Good: context stored in a struct field for later use
func (s *Store) Init(ctx context.Context) {
	s.ctx = ctx
	helper(1)
	helper(2)
}

// Bad: context parameter never referenced
func Unused(ctx context.Context, n int) int { // want `context parameter "ctx" is received but never used; pass it to sub-calls or remove it`
	return n * 2
}

// Bad: unnamed context parameter is treated as "ctx" and is never used
func Unnamed(context.Context, int) {} // want `context parameter "ctx" is received but never used`

// Bad: context explicitly discarded with _
func Ignored(_ context.Context, n int) int { // want `context parameter is explicitly ignored with '_'; this breaks tracing and cancellation propagation`
	return n
}

// Bad: _ context while the body makes calls (the blank identifier shows up
// in the body, which routes to the second message)
func IgnoredWithCalls(_ context.Context, xs []int) { // want `context parameter is explicitly ignored with '_'; HTTP/API calls in this function won't support tracing or cancellation`
	for _, x := range xs {
		helper(x)
	}
	helper(0)
	helper(1)
}

// Bad: context referenced but never handed to the calls made here
func NotPropagated(ctx context.Context) { // want `context parameter "ctx" is not passed to any sub-function calls; ensure context is propagated for tracing/cancellation`
	_ = ctx
	helper(1)
	helper(2)
}

// Good: context referenced in a short function
func ShortRef(ctx context.Context) {
	_ = ctx
	helper(1)
}

// Good: context referenced in a longer function without calls
func NoCalls(ctx context.Context) int {
	_ = ctx
	x := 1
	x++
	return x
}

// Bad: context.Background/TODO while a context is available
func UsesBackground(ctx context.Context) {
	helper(ctx)
	_ = context.Background() // want `context.Background\(\) used when context parameter is available; use the passed context instead`
	_ = context.TODO()       // want `context.TODO\(\) used when context parameter is available; use the passed context instead`
	c, cancel := context.WithCancel(ctx)
	defer cancel()
	_ = c
}

// Bad: package-level calls that have context-aware variants
func PackageCalls(ctx context.Context) {
	helper(ctx)
	_, _ = http.Get("http://x")                     // want `http.Get called without context; use http.NewRequestWithContext and client.Do instead`
	_, _ = http.Post("http://x", "text/plain", nil) // want `http.Post called without context`
	_, _ = http.PostForm("http://x", url.Values{})  // want `http.PostForm called without context`
	_, _ = http.Head("http://x")                    // want `http.Head called without context`
	_ = exec.Command("ls")                          // want `exec.Command called without context; use exec.CommandContext instead`
	_, _ = grpc.Dial("x")                           // want `grpc.Dial called without context; use grpc.DialContext instead`
}

// Good: the same calls without a context parameter are not flagged
func PackageCallsNoContext() {
	_, _ = http.Get("http://x")
	_ = exec.Command("ls")
}

// Bad: http.NewRequest is flagged with or without a context parameter
func NewRequestNoContext() {
	_, _ = http.NewRequest("GET", "http://x", nil) // want `http.NewRequest is deprecated in favor of http.NewRequestWithContext`
}

func NewRequestWithCtx(ctx context.Context) {
	helper(ctx)
	_, _ = http.NewRequest("GET", "http://x", nil) // want `http.NewRequest is deprecated in favor of http.NewRequestWithContext`
}

// Bad: time.Sleep while a context is available
func Sleeps(ctx context.Context) {
	helper(ctx)
	time.Sleep(time.Second) // want `time.Sleep called when context is available; use select with <-ctx.Done\(\) and time.After\(\) instead`
}

// Good: time.Sleep without a context parameter
func SleepsNoContext() {
	time.Sleep(time.Second)
}

// Good: methods with Context variants called with a context-like first argument
func (s *Store) Queries(reqCtx context.Context, r *http.Request) {
	s.db.Query(reqCtx, "SELECT 1")
	s.db.QueryRow(context.Background(), "SELECT 1") // want `context.Background\(\) used when context parameter is available`
	s.getDB().Exec(r.Context(), "DELETE")
	s.dbs[0].Prepare(withContext(reqCtx))
	s.sub.inner.db.Begin(reqCtx)
	func() {}()
}

// Good: a variable literally named ctx is accepted as a context argument
func (s *Store) NamedCtx(parent context.Context) {
	ctx := parent
	s.db.Exec(ctx, "UPDATE")
}

func withContext(ctx context.Context) context.Context { return ctx }

// Suppressed via nolint: method calls without a context argument
func (s *Store) Suppressed(ctx context.Context) {
	helper(ctx)
	s.db.Query("SELECT 1") //nolint:contextpropagation
	s.db.Exec()            //nolint:golint-sl
}

// Good: exempt function names, even with a context parameter
func TestMain(ctx context.Context)      {}
func BenchmarkMain(ctx context.Context) {}
func TestHelper(ctx context.Context)    {}

// Good: mock receivers are skipped
type MockStore struct{}

func (m *MockStore) Get(ctx context.Context) {}

type mockStore struct{}

func (m mockStore) Get(ctx context.Context) {}

// Bad: a generic receiver is not a mock
type Box[T any] struct{}

func (b *Box[T]) Get(ctx context.Context) {} // want `context parameter "ctx" is received but never used`

// Good: nothing to check without a body
func external(ctx context.Context)
