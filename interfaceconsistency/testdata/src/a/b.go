package a

import "strings"

// Second file without interfaces: mock bookkeeping skips it.

var separator = strings.Repeat("-", 3)

func joinParts(parts ...string) string {
	return strings.Join(parts, separator)
}
