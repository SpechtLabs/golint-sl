package a

import (
	"context"
	"sync"
)

func work() {}

// Done is a package-level function; a bare Done() call counts as cleanup.
func Done() {}

type MyContext struct{}

type worker struct{ wg sync.WaitGroup }

// Bad: infinite loop with no way to stop it
func Spin() {
	go func() { // want `goroutine with infinite loop has no way to stop; add select with <-ctx.Done\(\) or done channel`
		for {
			work()
		}
	}()
}

// Bad: inside a function with a context, an infinite loop without cleanup
// gets both diagnostics
func SpinWithContext(ctx context.Context) {
	go func() { // want `goroutine with infinite loop has no way to stop` `goroutine spawned without cleanup mechanism; consider passing context and checking ctx.Done\(\), or use sync.WaitGroup`
		for {
			work()
		}
	}()
}

// Bad: a function with a context spawns a goroutine without any cleanup
func Fire(ctx context.Context) {
	go func() { // want `goroutine spawned without cleanup mechanism`
		work()
	}()
}

// Bad: any parameter type containing "Context" counts as a context
func FireCustom(c MyContext) {
	go func() { // want `goroutine spawned without cleanup mechanism`
		work()
	}()
}

// Good: infinite loop that selects on ctx.Done()
func LoopWithContext(ctx context.Context, ch chan int) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case v := <-ch:
				_ = v
			}
		}
	}()
}

// Good: infinite loop that selects on a done channel
func LoopWithDoneChannel(done chan struct{}, out chan int) {
	go func() {
		for {
			select {
			case <-done:
				return
			case out <- 1:
			default:
			}
		}
	}()
}

// Good: a done channel received through an assignment
func LoopWithAssignedDone(ctx context.Context, stopDone chan struct{}) {
	go func() {
		for {
			select {
			case _, ok := <-stopDone:
				if !ok {
					return
				}
			}
		}
	}()
}

// Good: a select without any done case still counts as no cleanup, but this
// function has no context and no infinite loop, so nothing is reported
func SelectWithoutDone(in, out chan int) {
	go func() {
		select {
		case v := <-in:
			out <- v
		}
	}()
}

// Good: WaitGroup.Done in a function with a context
func WithWaitGroup(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		work()
	}()
	wg.Wait()
}

// Good: Done called on a nested selector
func (w *worker) Run(ctx context.Context) {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		work()
	}()
}

// Good: a bare Done() call counts as cleanup
func WithBareDone(ctx context.Context) {
	go func() {
		defer Done()
		func() {}()
	}()
}

// Good: blocking on ctx.Done() outside a select counts as a Done call
func WaitForCancel(ctx context.Context) {
	go func() {
		<-ctx.Done()
		work()
	}()
}

// Good: without a context parameter, bounded goroutines are not reported
func Bounded(ch chan int) {
	go func() {
		for i := 0; i < 3; i++ {
			work()
		}
		for v := range ch {
			_ = v
		}
		<-ch
	}()
}

// Good: go statements calling named functions are not analyzed
func Named(ctx context.Context) {
	go work()
}

// Good: nolint suppresses the diagnostic
func Suppressed(ctx context.Context) {
	go func() { //nolint:goroutineleak
		work()
	}()

	//nolint:golint-sl
	go func() {
		for {
			work()
		}
	}()
}

// Bad: nolint for another analyzer does not suppress
func OtherLinter(ctx context.Context) {
	go func() { //nolint:nilcheck // want `goroutine spawned without cleanup mechanism`
		work()
	}()
}
