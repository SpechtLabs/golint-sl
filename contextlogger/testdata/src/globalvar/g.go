// Package globalvar keeps its logger in a package-level variable named log.
package globalvar

import (
	"context"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// Bad: a method on the package-level log variable is a global logger call
func Handle(ctx context.Context) {
	log.Info("x") // want `function has context parameter but uses global logger`
}
