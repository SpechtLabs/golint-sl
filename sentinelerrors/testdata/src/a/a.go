// Package a holds the main sentinelerrors cases.
package a

import (
	"errors"
	"fmt"
	"strings"
)

const constMessage = "constant message"

// Good: a package-level sentinel declared before any function.
var ErrFirst = errors.New("first")

// Good: a grouped block of package-level sentinels.
var (
	ErrNotFound     = errors.New("item not found")
	ErrInvalidInput = errors.New("invalid input")
)

// Good: a package-level fmt.Errorf is a sentinel too.
var ErrPlain = fmt.Errorf("plain sentinel")

type store struct{}

// Bad: inline errors.New with a literal message.
func Lookup(id string) error {
	if id == "" {
		return errors.New("invalid input") // want `inline errors.New\(\) in function "Lookup"; define a package-level sentinel error`
	}
	return nil
}

// Bad: inline errors.New inside a method names the method.
func (s *store) Get() error {
	return errors.New("not found") // want `inline errors.New\(\) in function "Get"`
}

// Bad: a constant identifier is not dynamic content.
func FromConst() error {
	return errors.New(constMessage) // want `inline errors.New\(\) in function "FromConst"`
}

// Bad: a closure is attributed to its enclosing function.
func WithClosure() func() error {
	return func() error {
		return errors.New("from closure") // want `inline errors.New\(\) in function "WithClosure"`
	}
}

// Bad: a local variable makes the message dynamic.
func DynamicLocal() error {
	msg := "computed"
	return errors.New(msg) // want `errors.New\(\) with dynamic content; use fmt.Errorf`
}

// Bad: a parameter in the message is dynamic content.
func DynamicParam(name string) error {
	return errors.New("bad name: " + name) // want `errors.New\(\) with dynamic content`
}

// Bad: a function call in the message is dynamic content.
func DynamicCall(n int) error {
	return errors.New(fmt.Sprintf("n=%d", n)) // want `errors.New\(\) with dynamic content`
}

// Bad: fmt.Errorf with a constant literal and nothing to format.
func ConstErrorf() error {
	return fmt.Errorf("plain message") // want `fmt.Errorf\(\) without %w verb and no formatting`
}

// Good: fmt.Errorf wrapping an error with %w.
func Wrap(err error) error {
	return fmt.Errorf("lookup failed: %w", err)
}

// Good: fmt.Errorf with formatting arguments.
func Formatted(n int) error {
	return fmt.Errorf("bad value %d", n)
}

// Good: fmt.Errorf with a non-literal format is left alone.
func NonLiteralFormat() error {
	return fmt.Errorf(constMessage)
}

// Good: calls that are not errors.New or fmt.Errorf are ignored, including
// plain function calls and selectors on non-identifiers.
func Unrelated() int {
	helper()
	_ = strings.ToUpper("x")
	return strings.NewReader("abc").Len()
}

func helper() {}

// Good: suppressed with a nolint directive.
func Suppressed() error {
	return errors.New("suppressed") //nolint:sentinelerrors
}

// Good: a package-level sentinel declared after a function is not attributed
// to that function.
var ErrAfter = errors.New("after")

// Bad: a function declared after the package-level sentinels is still checked.
func Late() error {
	return errors.New("late") // want `inline errors.New\(\) in function "Late"`
}
