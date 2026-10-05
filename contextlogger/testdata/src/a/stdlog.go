package a

import (
	"context"
	"log"
)

// Bad: the standard library's global logger
func StdLog(ctx context.Context) {
	log.Print("x") // want `function has context parameter but uses global logger`
}
