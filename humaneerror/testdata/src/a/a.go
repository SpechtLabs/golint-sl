package a

import (
	"errors"
	"fmt"

	humane "github.com/sierrasoftworks/humane-errors-go"
)

// Good: Exported function returns humane.Error
func GoodFunction() humane.Error {
	return nil
}

// Bad: Exported function returns plain error
func BadFunction() error { // want `exported function "BadFunction" returns plain 'error'; use 'humane.Error'`
	return nil
}

// Good: unexported function can return plain error (internal use)
func internalFunction() error {
	return nil
}

// Good: humane.New with advice
func GoodNew() humane.Error {
	return humane.New("something failed", "try restarting the service")
}

// Bad: humane.New without advice
func BadNew() humane.Error {
	return humane.New("something failed") // want `humane.New\(\) should include at least one advice string`
}

// Good: humane.Wrap with advice
func GoodWrap(err error) humane.Error {
	return humane.Wrap(err, "failed to process", "check the input format")
}

// Bad: humane.Wrap without advice
func BadWrap(err error) humane.Error {
	return humane.Wrap(err, "failed to process") // want `humane.Wrap\(\) should include at least one advice string`
}

// Bad: using errors.New
func UsingErrorsNew() error { // want `exported function "UsingErrorsNew" returns plain 'error'`
	return errors.New("bad") // want `avoid errors.New\(\); use humane.New`
}

// Bad: using fmt.Errorf
func UsingFmtErrorf() error { // want `exported function "UsingFmtErrorf" returns plain 'error'`
	return fmt.Errorf("bad: %w", errors.New("x")) // want `avoid fmt.Errorf\(\); use humane.Wrap` `avoid errors.New\(\); use humane.New`
}

// Test functions are exempt
func TestSomething() error {
	return errors.New("test error") // want `avoid errors.New\(\); use humane.New`
}

// Bad: humane.Wrap with only error message, no advice (common mistake)
func BadWrapNoAdvice(e error) humane.Error {
	return humane.Wrap(e, "Failed to parse validityPeriod") // want `humane.Wrap\(\) should include at least one advice string`
}

// Good: humane.Wrap with proper advice
func GoodWrapWithAdvice(e error) humane.Error {
	return humane.Wrap(e, "Failed to parse validityPeriod", "Ensure the validity period is in ISO 8601 format (e.g., P1Y for 1 year)")
}

// Bad: Multiple Wrap calls without advice in same function
func MultipleBadWraps(e error) humane.Error {
	if e != nil {
		return humane.Wrap(e, "first error") // want `humane.Wrap\(\) should include at least one advice string`
	}
	return humane.Wrap(e, "second error") // want `humane.Wrap\(\) should include at least one advice string`
}

// Good: humane.Newf with formatted message and WithAdvice option.
// The format placeholder %q is NOT advice and must not be flagged.
func GoodNewf(name string) humane.Error {
	return humane.Newf("user %q not found", name,
		humane.WithAdvice("Verify the user exists in the directory"))
}

// Bad: humane.Newf without any WithAdvice option.
func BadNewfNoAdvice(name string) humane.Error {
	return humane.Newf("user %q not found", name) // want `humane.Newf\(\) should include at least one humane.WithAdvice`
}

// Good: humane.Wrapf with formatted message and WithAdvice option.
// The format string itself is NOT advice and must not be flagged.
func GoodWrapf(e error, ref string) humane.Error {
	return humane.Wrapf(e, "failed to fetch %s", ref,
		humane.WithAdvice("Verify the Roadie API is reachable"))
}

// Bad: humane.Wrapf without any WithAdvice option.
func BadWrapfNoAdvice(e error, ref string) humane.Error {
	return humane.Wrapf(e, "failed to fetch %s", ref) // want `humane.Wrapf\(\) should include at least one humane.WithAdvice`
}

// Good: WithAdvice with multiple actionable strings.
func GoodWrapfMultiAdvice(e error, name string) humane.Error {
	return humane.Wrapf(e, "config %q invalid", name,
		humane.WithAdvice(
			"Verify the YAML is well-formed",
			"Check the schema against docs/reference",
		))
}

// Bad: WithAdvice with a non-actionable string is still flagged via
// checkAdviceQuality (the advice itself is too vague).
func BadWrapfVagueAdvice(e error) humane.Error {
	return humane.Wrapf(e, "operation %s failed", "x",
		humane.WithAdvice("failed")) // want `advice "failed" may not be actionable`
}

// --- Exemptions for exported functions ---

// Good: exported functions without results are not checked.
func NoResults() {}

// Good: Benchmark* functions are skipped like Test* functions.
func BenchmarkSomething() error {
	return nil
}

// Writer implements io.Writer.
type Writer struct{}

// Good: methods implementing stdlib interfaces must return plain error.
func (w *Writer) Write(p []byte) (int, error) {
	return len(p), nil
}

// Good: standalone functions with a stdlib interface method name are exempt
// too, and fmt.Errorf is allowed inside them.
func Close() error {
	return fmt.Errorf("closing: %w", errClosed)
}

// --- Framework callbacks may use fmt.Errorf ---

// Good: functions whose name matches a callback pattern may use fmt.Errorf.
func requestHandler() error {
	return fmt.Errorf("handling: %w", errClosed)
}

// Bad: errors.New is flagged even inside framework callbacks.
var errClosed = errors.New("closed") // want `avoid errors.New\(\); use humane.New`

// --- Calls the analyzer ignores ---

type options struct {
	adv adviser
}

type adviser struct{}

func (adviser) WithAdvice(advice ...string) humane.Option { return nil }

func describe(s string) string { return s }

// Good: plain function calls and calls on non-identifier receivers are not
// humane constructors.
func plainCalls(o options) {
	_ = describe("x")
	_ = o.adv.WithAdvice("not a humane call")
}

// Bad: the advice check is syntactic; WithAdvice options that are not called
// on the humane package identifier (here a struct field and a local
// variable) do not count, and other calls in the format args are not advice.
func NewfWithForeignOptions(o options, name string) humane.Error {
	adv := o.adv
	return humane.Newf("user %q not found", describe(name), fmt.Sprint(name), o.adv.WithAdvice("x"), adv.WithAdvice("y"), name) // want `humane.Newf\(\) should include at least one humane.WithAdvice`
}

// Bad: humane.Wrapf with fewer than two arguments has no room for advice.
func WrapfTooFewArgs(e error) humane.Error {
	return humane.Wrapf(e, "failed") // want `humane.Wrapf\(\) should include at least one humane.WithAdvice`
}

// Bad: non-actionable advice in humane.New.
func NewVagueAdvice() humane.Error {
	return humane.New("db unreachable", "check error") // want `advice "check error" may not be actionable`
}

// Good: long advice is accepted even if it contains a vague phrase.
func NewLongAdvice() humane.Error {
	return humane.New("db unreachable", "If the connection failed, verify that DATABASE_URL points at a running server")
}

// --- nolint suppression ---

// Good: diagnostics silenced by nolint directives.
func Suppressed() error { //nolint:humaneerror
	_ = errors.New("inline") //nolint:humaneerror

	//nolint:golint-sl
	_ = errors.New("preceding line")

	_ = errors.New("other analyzer") //nolint:wideevents // want `avoid errors.New\(\); use humane.New`
	return nil
}
