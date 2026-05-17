// Package pluginlog provides log-level helpers for the plugin server side.
package pluginlog

import (
	"os"

	hclog "github.com/hashicorp/go-hclog"
)

// Level returns the hclog level to use for the plugin server's go-plugin logger.
// It reads SWM_LOG_LEVEL set by the host process; falls back to hclog.Warn so
// plugin debug noise is suppressed by default even without explicit host support.
func Level() hclog.Level {
	if v := os.Getenv("SWM_LOG_LEVEL"); v != "" {
		if l := hclog.LevelFromString(v); l != hclog.NoLevel {
			return l
		}
	}

	return hclog.Warn
}
