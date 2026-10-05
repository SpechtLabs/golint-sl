// Package a holds the sideeffects analyzer cases.
package a

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type local struct{}

func (local) Do() {}

func helper() {}

// --- reconcilers ---------------------------------------------------------

type FooReconciler struct {
	client *http.Client
	db     *sql.DB
	fetch  func() error
	runner interface{ Run() }
}

// Bad: direct HTTP and database calls from a reconciler.
func (r *FooReconciler) Reconcile(ctx context.Context, req *http.Request) error {
	_, _ = http.Get("https://example.com")                     // want `reconciler should not make direct net/http.Get call; use service layer abstraction` `reconciler should not make HTTP calls directly; inject an HTTP client interface`
	_, _ = http.Post("https://example.com", "text/plain", nil) // want `direct net/http.Post call` `should not make HTTP calls directly`
	_, _ = http.Head("https://example.com")                    // want `reconciler should not make HTTP calls directly`
	_, _ = r.client.Do(req)                                    // want `direct net/http.Do call` `should not make HTTP calls directly`
	_, _ = sql.Open("postgres", "dsn")                         // want `direct database/sql.Open call` `reconciler should not access database directly; use repository pattern`
	_, _ = r.db.Exec("DELETE FROM t")                          // want `reconciler should not access database directly`
	return nil
}

// Good: building a request, local helpers, method expressions on project
// types, dynamic and interface calls are not side effects of interest.
func (r *FooReconciler) prepare(ctx context.Context) error {
	_, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.com", nil)
	helper()
	local.Do(local{})
	r.runner.Run()
	_ = r.fetch()
	return err
}

type BarController struct{}

// Bad: any method on a controller type is treated as reconciler code.
func (c BarController) Sync() {
	_, _ = http.Get("https://example.com") // want `direct net/http.Get call` `should not make HTTP calls directly`
}

type Syncer struct{ client *http.Client }

// Bad: a method named Reconcile is a reconciler whatever its receiver.
func (s *Syncer) Reconcile(req *http.Request) {
	_, _ = s.client.Do(req) // want `direct net/http.Do call` `should not make HTTP calls directly`
}

// Good: other methods on non-reconciler types are not checked.
func (s *Syncer) Fetch(req *http.Request) {
	_, _ = s.client.Do(req)
}

// Good: a plain function named Reconcile has no receiver.
func Reconcile() {
	_, _ = http.Get("https://example.com")
}

// Good: suppressed with a nolint directive.
func (r *FooReconciler) Probe() {
	_, _ = http.Get("https://example.com") //nolint:sideeffects
}

// --- pure functions ------------------------------------------------------

// Bad: a parse function reading a file.
func parseConfig(path string) ([]byte, error) {
	return os.ReadFile(path) // want `function "parseConfig" should be pure but contains I/O operation ReadFile`
}

// Bad: I/O helpers from io are I/O too.
func ValidateBody(r io.Reader) error {
	_, err := io.ReadAll(r) // want `function "ValidateBody" should be pure but contains I/O operation ReadAll`
	return err
}

// Bad: time-dependent pure functions.
func computeDeadline(d time.Duration) time.Time {
	return time.Now().Add(d) // want `function "computeDeadline" should be pure but depends on time; accept time as parameter instead`
}

func calculateAge(t time.Time) (time.Duration, time.Duration) {
	return time.Since(t), time.Until(t) // want `function "calculateAge" should be pure but depends on time` `function "calculateAge" should be pure but depends on time`
}

// Good: pure functions without I/O, including calls into I/O packages that
// do no I/O, time helpers that do not read the clock, interface calls and
// method expressions.
func formatName(first, last string, err error, r io.Reader, at int64) string {
	_ = os.IsNotExist(err)
	_ = time.Unix(at, 0)
	_, _ = r.Read(nil)
	local.Do(local{})
	return strings.ToUpper(first) + " " + last
}

func convertUnits(v float64) float64 { return v * 2.54 }

func defaultValidatorRules() []string { return []string{"required"} }

// Good: suppressed with a nolint directive.
func parseSuppressed(path string) ([]byte, error) {
	return os.ReadFile(path) //nolint:sideeffects
}

// Good: functions with other names may do I/O.
func loadConfig(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// --- handlers ------------------------------------------------------------

var (
	requestCount int
	lastPath     string
	settings     = struct{ debug bool }{}
)

// Bad: an HTTP handler mutating package state.
func countRequests(w http.ResponseWriter, r *http.Request) {
	requestCount++ // want `handler function should not mutate global state "requestCount"; use dependency injection`
}

// Bad: a gin handler mutating package state.
func remember(c *gin.Context, path string) {
	lastPath = path // want `handler function should not mutate global state "lastPath"`
	c.JSON(200, nil)
}

// Good: handlers that only read globals or write locals and parameters.
func status(w http.ResponseWriter, r *http.Request) {
	n := requestCount
	inc := func() { n++ }
	inc()
	r.Method = "GET"
	_, _ = w.Write([]byte(lastPath))
}

// Good: functions that are not handlers may mutate globals.
func reset() {
	requestCount = 0
}

// Good: suppressed with a nolint directive.
func countSuppressed(w http.ResponseWriter, r *http.Request) {
	requestCount++ //nolint:sideeffects
}
