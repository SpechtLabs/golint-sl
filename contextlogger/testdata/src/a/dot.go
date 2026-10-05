package a

import (
	. "context"

	"example.com/log"
)

// Bad: a dot-imported Context still counts as a context parameter
func DotImported(ctx Context) {
	log.Info("x") // want `function has context parameter but uses global logger`
}
