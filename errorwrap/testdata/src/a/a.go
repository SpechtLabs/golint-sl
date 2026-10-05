package a

import (
	"errors"
	"fmt"

	pkgerrors "github.com/pkg/errors"
	humane "github.com/sierrasoftworks/humane-errors-go"
)

func step() error              { return nil }
func pair() (int, error)       { return 0, nil }
func humaneStep() humane.Error { return nil }

// Bad: err is returned without context
func BareReturn() error {
	err := step()
	if err != nil {
		return err // want `returning error "err" without wrapping; add context with humane.Wrap\(err, message, advice...\)`
	}
	_ = step()
	return nil
}

// Bad: bare err in a multi-value return
func BareMultiReturn() (int, error) {
	n, err := pair()
	if err != nil {
		return 0, err // want `returning error "err" without wrapping`
	}
	_ = step()
	return n, nil
}

// Bad: assigned variables ending in Err or Error are tracked too
func BareNamedErr() error {
	loadErr := step()
	parseError := step()
	if loadErr != nil {
		return loadErr // want `returning error "loadErr" without wrapping`
	}
	return parseError // want `returning error "parseError" without wrapping`
}

// Bad: parameters named err or *Err are treated as errors even if never assigned
func ForwardParam(err error, readErr error) error {
	_ = step()
	_ = step()
	if readErr != nil {
		return readErr // want `returning error "readErr" without wrapping`
	}
	return err // want `returning error "err" without wrapping`
}

// Good: parameters ending in Error are only tracked when assigned in the body
func ForwardErrorParam(lastError error) error {
	_ = step()
	_ = step()
	return lastError
}

// Bad: fmt.Errorf without %w does not count as wrapping
func ErrorfWithoutW() error {
	err := step()
	if err != nil {
		err = fmt.Errorf("step failed: %v", err)
		return err // want `returning error "err" without wrapping`
	}
	return nil
}

// Bad: fmt.Errorf with a non-literal format does not count as wrapping
func ErrorfDynamicFormat(format string) error {
	err := step()
	if err != nil {
		err = fmt.Errorf(format, err)
		return err // want `returning error "err" without wrapping`
	}
	return nil
}

type logger struct{}

func (logger) Errorf() error { return nil }

// Bad: an Errorf method without arguments does not wrap anything
func ErrorfNoArgs(l logger) error {
	err := step()
	if err != nil {
		_ = l.Errorf()
		return err // want `returning error "err" without wrapping`
	}
	return nil
}

// Good: err was rewrapped with %w before it is returned
func WrappedWithW() error {
	err := step()
	if err != nil {
		err = fmt.Errorf("step failed: %w", err)
		return err
	}
	_ = step()
	return nil
}

// Good: wrapping helpers mark the error as wrapped
func WrappedWithHelpers() error {
	err := step()
	if err != nil {
		err = pkgerrors.Wrap(err, "step")
		return err
	}
	wrapfErr := step()
	if wrapfErr != nil {
		wrapfErr = pkgerrors.Wrapf(wrapfErr, "step %d", 2)
		return wrapfErr
	}
	msgErr := step()
	if msgErr != nil {
		msgErr = pkgerrors.WithMessage(msgErr, "step")
		return msgErr
	}
	return nil
}

// Good: humane.Wrap marks the error as wrapped
func WrappedWithHumane() error {
	err := step()
	if err != nil {
		err = humane.Wrap(err, "step failed", "retry later")
		return err
	}
	return nil
}

// Errorf is a package-local helper; calls to a bare Errorf count as wrapping.
func Errorf(format string, args ...any) error { return nil }

// Good: a bare Errorf identifier call counts as wrapping
func WrappedWithLocalErrorf() error {
	err := step()
	if err != nil {
		err = Errorf("step: %v", err)
		return err
	}
	return nil
}

type factory struct{}

func (factory) New(msg string) error { return nil }

// Good: returning a constructed error is never a bare return
func NewErrors(f factory) error {
	if err := step(); err != nil {
		return humane.New("step failed", "retry later")
	}
	if err := step(); err != nil {
		return errors.New("step failed")
	}
	if err := step(); err != nil {
		return f.New("step failed")
	}
	return nil
}

// Good: functions returning humane.Error are skipped entirely
func HumaneResult() humane.Error {
	err := humaneStep()
	if err != nil {
		return err
	}
	_ = step()
	return nil
}

// Good: Test-prefixed functions are skipped
func TestLikeHelper() error {
	err := step()
	if err != nil {
		return err
	}
	_ = step()
	return nil
}

// Good: functions with at most two statements are skipped
func short() error {
	err := step()
	return err
}

// Good: functions with fewer than two meaningful operations are skipped
func fewOperations(x int) error {
	err := step()
	if x > 0 {
		return err
	}
	return nil
}

type state struct{ n int }

// Good: naked returns carry no result expressions
func noResults(s *state) {
	s.n = 1
	a, b := 1, 2
	s.n = a + b
	if s.n > 2 {
		return
	}
}

// Good: nil and non-identifier results are not bare error returns
func nonIdentResults() (int, error) {
	_ = step()
	_ = step()
	if false {
		return len("x"), errors.New("x")
	}
	return 0, nil
}

// Good: nolint suppresses the diagnostic
func Suppressed() error {
	err := step()
	if err != nil {
		return err //nolint:errorwrap
	}
	_ = step()
	return nil
}

// Good: a nolint on the preceding line suppresses the diagnostic
func SuppressedAbove() error {
	err := step()
	if err != nil {
		//nolint:golint-sl
		return err
	}
	_ = step()
	return nil
}

// Bad: field writes count as meaningful operations
func BareAfterFieldWrite(s *state) error {
	s.n = 1
	err := step()
	if err != nil {
		return err // want `returning error "err" without wrapping`
	}
	return nil
}

// Good: functions without a body (implemented elsewhere) are skipped
func external() error
