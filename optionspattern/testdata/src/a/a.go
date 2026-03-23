package a

import "context"

// --- Option type definitions ---

// ServerOption is an option for configuring a Server.
type ServerOption func(*serverConfig)

type serverConfig struct {
	timeout int
	debug   bool
}

// --- Good: With* functions returning Option ---

// WithTimeout sets the timeout.
func WithTimeout(t int) ServerOption {
	return func(c *serverConfig) {
		c.timeout = t
	}
}

// WithDebug enables debug mode.
func WithDebug() ServerOption {
	return func(c *serverConfig) {
		c.debug = true
	}
}

// --- Good: Builder pattern (method returns same receiver type) ---

// Builder constructs things.
type Builder struct {
	prefix  string
	service string
	tier    string
}

// NewBuilder creates a new Builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// WithPrefix sets the prefix — builder pattern, should NOT be flagged.
func (b *Builder) WithPrefix(prefix string) *Builder {
	b.prefix = prefix
	return b
}

// WithService sets the service — builder pattern, should NOT be flagged.
func (b *Builder) WithService(service string) *Builder {
	b.service = service
	return b
}

// WithTier sets the tier — builder pattern, should NOT be flagged.
func (b *Builder) WithTier(tier string) *Builder {
	b.tier = tier
	return b
}

// --- Good: Context enrichment (takes and returns context.Context) ---

// WithFields enriches context — context enrichment, should NOT be flagged.
func WithFields(ctx context.Context, key, value string) context.Context {
	return ctx
}

// WithTraceContext adds trace IDs — context enrichment, should NOT be flagged.
func WithTraceContext(ctx context.Context) context.Context {
	return ctx
}

// --- Bad: With* functions that don't return Option, not builder, not context ---

// WithBadNaming starts with With but returns string.
func WithBadNaming() string { // want `function "WithBadNaming" starts with 'With' but doesn't return an Option type`
	return "bad"
}

// WithAlsoBad returns int, not Option.
func WithAlsoBad(x int) int { // want `function "WithAlsoBad" starts with 'With' but doesn't return an Option type`
	return x
}

// --- Good: Constructor with options ---

// NewServer creates a new server with options.
func NewServer(opts ...ServerOption) *serverConfig {
	cfg := &serverConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// --- Bad: Constructor with too many params ---

// NewBigConstructor has too many params.
func NewBigConstructor(a, b, c, d, e int) *serverConfig { // want `constructor "NewBigConstructor" has 5 parameters`
	return &serverConfig{}
}

// --- Good: private with* functions are skipped ---

func withPrivate() string {
	return ""
}
