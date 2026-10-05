package a

import (
	"context"

	"gin"
)

type Request struct{}

type Service struct{}

// Good: context is the first parameter
func ProcessRequest(ctx context.Context, req *Request) error {
	return nil
}

// Good: method with context first
func (s *Service) Handle(ctx context.Context, id string) (string, error) {
	return id, nil
}

// Good: no parameters at all
func NoParams() {}

// Good: a single parameter can't be out of order
func OnlyContext(ctx context.Context) {}

// Good: several parameters, none of them a context
func NoContext(a int, b string) {}

// Good: a func-typed parameter mentioning context is not itself a context
func CallbackFirst(cb func(context.Context), n int) {}

// Bad: context is the second parameter
func BadOrder(req *Request, ctx context.Context) error { // want `context.Context should be the first parameter in BadOrder, not parameter 2`
	return nil
}

// Bad: method with context second
func (s *Service) BadHandle(id string, ctx context.Context) error { // want `context.Context should be the first parameter in BadHandle, not parameter 2`
	return nil
}

// Bad: context third, after a callback that mentions context in its type
func BadAfterCallback(cb func(context.Context), n int, ctx context.Context) { // want `context.Context should be the first parameter in BadAfterCallback, not parameter 3`
}

// Suppressed: nolint on the preceding line
//
//nolint:contextfirst
func SuppressedBefore(id string, ctx context.Context) {}

func SuppressedInline(id string, ctx context.Context) {} //nolint:golint-sl

func Literals() {
	// Good: anonymous function with context first
	_ = func(ctx context.Context, n int) {}

	// Good: anonymous function with a single parameter
	_ = func(ctx context.Context) {}

	// Bad: anonymous function with context second
	_ = func(n int, ctx context.Context) {} // want `context.Context should be the first parameter in anonymous function, not parameter 2`
}

// Ctx is an alias of context.Context, which is still a context.
type Ctx = context.Context

// Bad: grouped parameters count one position per name
func Grouped(a, b int, ctx context.Context) {} // want `context.Context should be the first parameter in Grouped, not parameter 3`

// Bad: a grouped field after an unnamed one
func GroupedAfter(req *Request, a, b int, ctx context.Context) {} // want `context.Context should be the first parameter in GroupedAfter, not parameter 4`

// Bad: an alias of context.Context is a context
func AliasSecond(id string, ctx Ctx) {} // want `context.Context should be the first parameter in AliasSecond, not parameter 2`

// Good: a framework type named Context is not a context.Context
func FrameworkContext(id string, c *gin.Context) {}

// Good: two contexts grouped first
func TwoContexts(parent, child context.Context, n int) {}
