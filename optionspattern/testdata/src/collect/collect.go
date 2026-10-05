// Package collect is input for AnalyzeOptionPatterns.
package collect

// ClientOption configures a client.
type ClientOption func(*client)

// OptionSet is a non-function type containing "Option".
type OptionSet struct{}

type client struct{}

// NewClient is a constructor.
func NewClient(opts ...ClientOption) *client { return &client{} }

// WithRetry is an option function.
func WithRetry() ClientOption { return func(*client) {} }

func (c *client) Close() {}
