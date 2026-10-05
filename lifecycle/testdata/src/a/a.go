package a

import "context"

// Good: Run takes a context, selects on ctx.Done(), and the type has Close.
type goodServer struct{ events chan int }

func (s *goodServer) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.events:
		}
	}
}

func (s *goodServer) Close() error { return nil }

// Good: value receiver, Start without a loop, Shutdown as the stop method.
type valueWorker struct{}

func (w valueWorker) Start(ctx context.Context) error { return nil }

func (w valueWorker) Shutdown(ctx context.Context) error { return nil }

// Good: a stop method declared before the run method still counts.
type stopFirst struct{}

func (s *stopFirst) GracefulStop(ctx context.Context) error { return nil }

func (s *stopFirst) Serve(ctx context.Context) error { return nil }

// Good: a done channel taken from the context is recognised by "Done()".
type doneChan struct{ c context.Context }

func (d *doneChan) Run(ctx context.Context) error {
	for range 3 {
		select {
		case <-d.c.Done():
			return nil
		default:
		}
	}
	return nil
}

func (d *doneChan) Stop() {}

// Bad: Run without parameters, and the type has no stop method.
type noParams struct{}

func (n *noParams) Run() error { // want `Run\(\) should accept context.Context as first parameter for cancellation support` `type "noParams" has Run\(\) method but no Close\(\)/Stop\(\)/GracefulStop\(\) method`
	return nil
}

// Bad: first parameter is not a context.
type wrongParam struct{}

func (w *wrongParam) Start(addr string) error { // want `Start\(\) first parameter should be context.Context, got string`
	return nil
}

func (w *wrongParam) GracefulShutdown() {}

// Bad: Serve loops without ever checking ctx.Done().
type busyLoop struct{ ch chan int }

func (b *busyLoop) Serve(ctx context.Context) error { // want `Serve\(\) has a loop but doesn't check ctx.Done\(\)`
	for v := range b.ch {
		_ = v
	}
	return nil
}

func (b *busyLoop) Close() error { return nil }

// Bad: a select without a Done() case does not count; the default clause,
// a non-unary receive and an unrelated receive are all skipped.
type selectNoDone struct{ ch chan int }

func (s *selectNoDone) Run(ctx context.Context) error { // want `Run\(\) has a loop but doesn't check ctx.Done\(\)`
	for {
		select {
		case v := <-s.ch:
			_ = v
		case s.ch <- 1:
		case <-s.ch:
		default:
			return nil
		}
	}
}

func (s *selectNoDone) Close() error { return nil }

// Good: a bodyless method (implemented elsewhere) is only checked for its
// signature. Methods that are not lifecycle methods are ignored.
type helper struct{}

func (h *helper) Run(ctx context.Context) error { return nil }

func (h *helper) Close() error { return nil }

func (h *helper) Other() {}

// Good: plain functions named Run are not methods.
func Run() {}

// Good: the diagnostics on this method are suppressed by nolint.
type suppressed struct{}

//nolint:lifecycle
func (s *suppressed) Run() {}
