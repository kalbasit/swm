package session

import (
	"os"
	"strings"
)

// pluginInternalVars lists the SWM_* variables that are only meaningful inside a
// plugin subprocess and must not appear in user-facing processes (tmux sessions, hooks).
var pluginInternalVars = map[string]bool{ //nolint:gochecknoglobals // package-level constant set
	"SWM_HOST_SOCKET":         true,
	"SWM_LOG_LEVEL":           true,
	"SWM_PLUGIN_MAGIC_COOKIE": true,
}

// filteredEnv returns os.Environ() with all plugin-internal SWM_* variables removed.
func filteredEnv() []string {
	src := os.Environ()
	out := make([]string, 0, len(src))

	for _, e := range src {
		key, _, _ := strings.Cut(e, "=")
		if !pluginInternalVars[key] {
			out = append(out, e)
		}
	}

	return out
}
