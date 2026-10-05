package a

import "context"

// --- Option type definitions that break the pattern ---

// Bad: an Option type that is not a function type.
type StructOption struct { // want `Option type "StructOption" should be a function type: type StructOption func\(\*T\)`
	timeout int
}

// Bad: an Option function type without parameters.
type EmptyOption func() // want `Option function type should take exactly one parameter \(pointer to config struct\)`

// Bad: an Option function type with two parameters.
type PairOption func(*serverConfig, int) // want `Option function type should take exactly one parameter`

// Bad: an Option function type taking the config by value.
type ValueOption func(serverConfig) // want `Option function parameter should be a pointer type \(\*T\)`

// Bad: an Option function type that returns a value.
type CheckedOption func(*serverConfig) error // want `Option function should not return any values`

// Good: the bare name Option with the canonical shape.
type Option func(*serverConfig)

// Good: types that are not named *Option are ignored.
type Settings struct{}

// --- Option function naming ---

// Bad: returns an Option without a standard prefix.
func Timeout(t int) ServerOption { // want `function "Timeout" returns Option but doesn't use a standard option prefix \(With, Allow, Enable, Disable, Set\); rename to WithTimeout`
	return func(c *serverConfig) { c.timeout = t }
}

// Good: the other standard prefixes.
func AllowInsecure() ServerOption    { return func(c *serverConfig) {} }
func EnableDebug() ServerOption      { return func(c *serverConfig) { c.debug = true } }
func DisableDebug() ServerOption     { return func(c *serverConfig) { c.debug = false } }
func SetTimeout(t int) Option        { return func(c *serverConfig) { c.timeout = t } }
func DefaultOptions() []ServerOption { return []ServerOption{WithDebug()} }

// Good: option functions with validation before the return.
func WithValidatedTimeout(t int) ServerOption {
	if t < 0 {
		t = 0
	}
	return func(c *serverConfig) { c.timeout = t }
}

// Good: an option function that returns a named function.
func WithNamedFunc() ServerOption {
	return applyDefaults
}

func applyDefaults(c *serverConfig) {}

// Good: an option function whose single statement is not a return.
func WithUnimplemented() ServerOption {
	panic("not implemented")
}

// Good: an option function returning an option and an error.
func WithChecked(t int) (ServerOption, error) {
	return WithTimeout(t), nil
}

// Good: an option function without a body.
func WithExternal() ServerOption

// Good: exported functions without results are not option functions.
func Reset() {}

// --- With* functions that are neither options, builders nor context enrichment ---

// Bad: a method that returns a different type than its receiver.
func (b *Builder) WithName(name string) string { // want `function "WithName" starts with 'With' but doesn't return an Option type`
	return name
}

// Bad: takes a context but does not return one.
func WithDeadline(ctx context.Context) error { // want `function "WithDeadline" starts with 'With' but doesn't return an Option type`
	return ctx.Err()
}

// --- Constructor parameter counting ---

// Good: an Option slice parameter counts as options.
func NewFromSlice(a, b, c, d, e int, opts []ServerOption) *serverConfig {
	return &serverConfig{}
}

// Good: variadic option types matched through the collected Option types.
func NewWithBareOption(a, b, c, d, e int, opts ...Option) *serverConfig {
	return &serverConfig{}
}

// Bad: a variadic non-option parameter still counts as a regular parameter.
func NewVariadic(a, b, c, d int, rest ...string) *serverConfig { // want `constructor "NewVariadic" has 5 parameters`
	return &serverConfig{}
}

// Bad: unnamed parameters count one each.
func NewUnnamed(int, int, int, int, string) *serverConfig { // want `constructor "NewUnnamed" has 5 parameters`
	return &serverConfig{}
}

// Good: four parameters are within the limit.
func NewSmall(a, b, c, d int) *serverConfig {
	return &serverConfig{}
}

// Good: test helpers are skipped.
func NewTestFixture(a, b, c, d, e int) *serverConfig    { return &serverConfig{} }
func NewFixtureForTest(a, b, c, d, e int) *serverConfig { return &serverConfig{} }

// --- nolint suppression ---

// Good: diagnostics silenced by nolint directives.
func NewSuppressed(a, b, c, d, e int) *serverConfig { //nolint:optionspattern
	return &serverConfig{}
}

//nolint:golint-sl
func NewSuppressedAbove(a, b, c, d, e int) *serverConfig { return &serverConfig{} }

func NewOtherNolint(a, b, c, d, e int) *serverConfig { //nolint:humaneerror // want `constructor "NewOtherNolint" has 5 parameters`
	return &serverConfig{}
}
