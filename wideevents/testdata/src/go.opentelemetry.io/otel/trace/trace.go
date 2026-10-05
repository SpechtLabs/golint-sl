// Package trace is a minimal stub of go.opentelemetry.io/otel/trace for the
// wideevents tests.
package trace

import "context"

// Span is an OpenTelemetry span.
type Span interface {
	SetAttributes(kv ...any)
	SetStatus(code int, description string)
	AddEvent(name string)
	End()
}

// Tracer starts spans.
type Tracer interface {
	Start(ctx context.Context, name string) (context.Context, Span)
}

// SpanFromContext returns the span stored in ctx.
func SpanFromContext(ctx context.Context) Span { return nil }
