package a

import (
	"context"
	"time"
)

// Good: a range over a finite collection is not a long-running loop, and
// the method blocks on <-ctx.Done() outside any select.
type Pool struct{ children []func(context.Context) }

func (p *Pool) Run(ctx context.Context) error {
	for _, c := range p.children {
		go c(ctx)
	}
	<-ctx.Done()
	return ctx.Err()
}

func (p *Pool) Close() error { return nil }

// Good: counted and conditional loops are bounded by their condition.
type batcher struct{ items []int }

func (b *batcher) Start(ctx context.Context) error {
	for i := 0; i < len(b.items); i++ {
		_ = b.items[i]
	}
	for len(b.items) > 0 {
		b.items = b.items[1:]
	}
	for range 3 {
	}
	return nil
}

func (b *batcher) Stop() {}

// Good: ctx.Err() checked inside the loop observes cancellation.
type poller struct{}

func (p *poller) Run(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		time.Sleep(time.Second)
	}
}

func (p *poller) Close() error { return nil }

// Good: a done channel taken out of the context before the loop counts.
type doneVar struct{ ch chan int }

func (d *doneVar) Serve(ctx context.Context) error {
	done := ctx.Done()
	for {
		select {
		case <-done:
			return nil
		case <-d.ch:
		}
	}
}

func (d *doneVar) Close() error { return nil }

// Bad: ranging over a ticker channel never ends on its own.
type ticking struct{ t *time.Ticker }

func (t *ticking) Run(ctx context.Context) error { // want `Run\(\) has a loop but doesn't check ctx.Done\(\)`
	for range t.t.C {
	}
	return nil
}

func (t *ticking) Close() error { return nil }

// Bad: a WaitGroup's Done is not a done channel, and Err on something that
// is not a context is not a cancellation check.
type notCancellation struct{ e interface{ Err() error } }

type waiter interface{ Done() }

func (n *notCancellation) Run(ctx context.Context, w waiter) error { // want `Run\(\) has a loop but doesn't check ctx.Done\(\)`
	for {
		w.Done()
		_ = n.e.Err()
	}
}

func (n *notCancellation) Close() error { return nil }

// Bad: generic receivers are lifecycle components too.
type G[T any] struct{ v T }

func (g *G[T]) Run() { // want `Run\(\) should accept context.Context as first parameter` `Run\(\) has a loop but doesn't check ctx.Done\(\)` `type "G" has Run\(\) method but no Close\(\)/Stop\(\)/GracefulStop\(\) method`
	for {
	}
}

// Good: a generic type with a stop method on an instantiated-style receiver.
type H[K comparable, V any] struct{ m map[K]V }

func (h H[K, V]) Start(ctx context.Context) error { return nil }

func (h *H[K, V]) Shutdown() {}

// Bad: the message names the run method the type actually has.
type W struct{}

func (w *W) Start(ctx context.Context) error { // want `type "W" has Start\(\) method but no Close\(\)/Stop\(\)/GracefulStop\(\) method`
	return nil
}

// Good: a type implementing context.Context is accepted as the first
// parameter; a type merely named like one is not.
type reqContext struct{ context.Context }

type MyContext struct{}

type ctxParams struct{}

func (c *ctxParams) Run(ctx *reqContext) error { return nil }

func (c *ctxParams) Start(ctx MyContext) error { // want `Start\(\) first parameter should be context.Context, got MyContext`
	return nil
}

func (c *ctxParams) Close() error { return nil }
