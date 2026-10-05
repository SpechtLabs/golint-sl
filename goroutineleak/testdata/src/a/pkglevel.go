package a

import "context"

// WithCtx takes a context; the declarations after it don't.
func WithCtx(ctx context.Context) { _ = ctx }

// Good: a goroutine in a package-level initializer has no enclosing function
// with a context, whatever function is declared before it
var _ = func() int {
	go func() {
		work()
	}()
	return 0
}()

// Bad: an infinite loop is reported in a package-level initializer too
var _ = func() int {
	go func() { // want `goroutine with infinite loop has no way to stop`
		for {
			work()
		}
	}()
	return 0
}()

// Bad: "for true" is an infinite loop like "for"
func SpinTrue() {
	go func() { // want `goroutine with infinite loop has no way to stop; add select with <-ctx.Done\(\) or done channel`
		for true {
			work()
		}
	}()
}

const forever = true

// Bad: so is a loop on any constant that is true
func SpinConst() {
	go func() { // want `goroutine with infinite loop has no way to stop`
		for i := 0; forever; i++ {
			work()
		}
	}()
}

// Good: a loop on a constant false condition never runs
func NeverRuns() {
	go func() {
		for false {
			work()
		}
	}()
}

// EndsWithCtx is the last declaration of this file and takes a context.
func EndsWithCtx(ctx context.Context) { _ = ctx }
