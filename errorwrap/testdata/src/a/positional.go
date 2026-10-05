package a

import "fmt"

func wrapErr(err *error, msg string) {
	if *err != nil {
		*err = fmt.Errorf("%s: %w", msg, *err)
	}
}

// Bad: wrapping err in an earlier return doesn't wrap a later bare return
func ReusedAfterWrap() error {
	err := step()
	if err != nil {
		return fmt.Errorf("first: %w", err)
	}
	err = step()
	if err != nil {
		return err // want `returning error "err" without wrapping; add context with humane.Wrap\(err, message, advice...\)`
	}
	return nil
}

// Bad: a reassignment after the wrap undoes it
func ReassignedAfterWrap() error {
	err := step()
	err = fmt.Errorf("first: %w", err)
	_ = err
	err = step()
	if err != nil {
		return err // want `returning error "err" without wrapping`
	}
	return nil
}

// Good: a conditional wrap before the return counts
func ConditionalWrap() error {
	err := step()
	_ = step()
	if err != nil {
		err = fmt.Errorf("step: %w", err)
	}
	return err
}

// Good: an error variable set to nil before the return isn't an error
func ResetToNil() error {
	err := step()
	_ = step()
	if err != nil {
		err = nil
	}
	return err
}

// Bad: a naked return of a named err result inside "if err != nil"
func NakedReturn() (n int, err error) {
	n, err = pair()
	if err != nil {
		return // want `returning error "err" without wrapping`
	}
	_ = step()
	return
}

// Good: a naked return after the error check returns a nil err
func NakedReturnAfterCheck() (n int, err error) {
	n, err = pair()
	if err != nil {
		return 0, fmt.Errorf("pair: %w", err)
	}
	_ = step()
	return
}

// Good: a named err result wrapped before the naked return
func NakedReturnWrapped() (err error) {
	err = step()
	_ = step()
	if err != nil {
		err = fmt.Errorf("step: %w", err)
		return
	}
	return
}

// Good: a deferred function wraps the named result on every return
func DeferredWrap() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("deferred: %w", err)
		}
	}()
	err = step()
	if err != nil {
		return err
	}
	_ = step()
	return nil
}

// Good: a deferred helper receives the named result's address
func DeferredHelper() (err error) {
	defer wrapErr(&err, "helper")
	err = step()
	if err != nil {
		return
	}
	_ = step()
	return nil
}

// Good: an explicit return of a named err result that was never set
func UnsetNamedResult() (n int, err error) {
	n = 1
	_ = step()
	_ = step()
	return n, err
}

// Bad: a return in one switch case isn't wrapped by another case
func SwitchCases(k int) error {
	err := step()
	_ = step()
	switch k {
	case 1:
		err = fmt.Errorf("one: %w", err)
		return err
	case 2:
		return err // want `returning error "err" without wrapping`
	}
	return nil
}

// Bad: a wrap in the if branch doesn't reach a return in the else branch
func IfElseBranches(ok bool) error {
	err := step()
	_ = step()
	if ok {
		err = fmt.Errorf("ok: %w", err)
		_ = err
	} else {
		return err // want `returning error "err" without wrapping`
	}
	return nil
}

// Bad: a later retry overwrites the nil error before the return
func RetryLoop() error {
	var err error
	for range 3 {
		err = step()
		if err == nil {
			break
		}
	}
	_ = step()
	return err // want `returning error "err" without wrapping`
}

// Good: a closure's own naked return is checked against its own results
func ClosureNaked() error {
	f := func() (err error) {
		err = step()
		_ = step()
		if err != nil {
			err = fmt.Errorf("closure: %w", err)
			return
		}
		return
	}
	_ = step()
	return f()
}
