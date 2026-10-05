package a

import "strings"

// Second file without interfaces or dependencies.

var separator = strings.Repeat("-", 3)

func joinParts(parts ...string) string {
	return strings.Join(parts, separator)
}
