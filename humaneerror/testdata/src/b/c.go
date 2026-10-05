package b

import "strings"

// Good: a file without a humane import; strings.ToUpper is not a humane call.
func title(s string) string {
	return strings.ToUpper(s)
}
