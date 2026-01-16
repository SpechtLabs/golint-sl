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
