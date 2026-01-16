// Package version provides version information for golint-sl.
package version

import (
	"fmt"
	"runtime"
)

// Build information set by ldflags during build.
// These are set via -ldflags at build time.
var (
	Version   = "dev"
	Commit    = "unknown"
	Date      = "unknown"
	GoVersion = runtime.Version()
)

// Info returns formatted version information.
func Info() string {
	return fmt.Sprintf("golint-sl %s (commit: %s, built: %s, %s)",
		Version, Commit, Date, GoVersion)
}

// Short returns just the version string.
func Short() string {
	return Version
}
