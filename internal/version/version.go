package version

import (
	"fmt"
	"runtime"
)

var (
	// Values are overridden at build time via -ldflags.
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// Info returns a multi-line version summary for --version flags and logs.
func Info() string {
	return fmt.Sprintf("Version: %s\nCommit: %s\nBuildDate: %s\nGoVersion: %s\n", Version, Commit, BuildDate, runtime.Version())
}
