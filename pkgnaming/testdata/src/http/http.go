// Package http checks that the stutter match is case-insensitive.
package http

// Bad: http.HTTPClient stutters.
type HTTPClient struct{} // want `type http.HTTPClient stutters; consider renaming to http.Client`

// Good: no stutter.
type Client struct{}
