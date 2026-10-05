package a

import "strconv"

// --- Types that mention Option but are not functional options ---

// Options is a plain config struct.
type Options struct {
	Port int
}

// Good: returning a config struct is not returning an option, so no option
// prefix is needed.
func ParseOptions(s string) (*Options, error) {
	port, err := strconv.Atoi(s)
	if err != nil {
		return nil, err
	}
	return &Options{Port: port}, nil
}

// Good: same for a value.
func LoadOptions() Options {
	return Options{Port: 8080}
}

// OptionSet is a named collection, not an option.
type OptionSet map[string]string

// Good: the type name contains Option but does not end in it.
func BuildOptionSet() OptionSet {
	return OptionSet{}
}

// Good: a slice of options is not an individual option.
func ProductionOptions() []ServerOption {
	return []ServerOption{WithDebug()}
}

// Good: With is a fine prefix for functions returning an options struct or
// a slice of options.
func WithDefaults() []ServerOption { return nil }
func WithBase() Options            { return Options{} }

// --- Interface-based options (zap.Option, grpc.DialOption) ---

// DialOption is an interface option with an apply method.
type DialOption interface {
	apply(*serverConfig)
}

type dialFunc func(*serverConfig)

func (f dialFunc) apply(c *serverConfig) { f(c) }

// Good: interface options are a valid definition and are named correctly here.
func WithBlock() DialOption {
	return dialFunc(func(c *serverConfig) {})
}

// Bad: an interface option still needs an option prefix.
func Insecure() DialOption { // want `function "Insecure" returns Option but doesn't use a standard option prefix`
	return dialFunc(func(c *serverConfig) {})
}

// Bad: an interface named *Option without an apply method is not an option.
type ListenOption interface { // want `Option type "ListenOption" should be a function type`
	Port() int
}

// Good: an alias of an option type is checked where the aliased type is
// defined, not here.
type AliasOption = ServerOption

// Bad: an alias of an option type is still an option.
func Verbose() AliasOption { // want `function "Verbose" returns Option but doesn't use a standard option prefix`
	return func(c *serverConfig) { c.debug = true }
}
