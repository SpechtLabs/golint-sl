// Package gin is a stub of github.com/gin-gonic/gin for testing the
// sideeffects analyzer.
package gin

// Context carries request-scoped state.
type Context struct{}

// JSON writes a JSON response.
func (c *Context) JSON(code int, obj any) {}
