// fakehook logs its environment variables and stdin to a file for test inspection.
// Set FAKEHOOK_LOG to the path where output should be written.
// Set FAKEHOOK_EXIT to "1" to simulate a failing hook.
package main

import (
	"io"
	"os"
)

func main() {
	if os.Getenv("FAKEHOOK_EXIT") == "1" {
		os.Exit(1)
	}

	logPath := os.Getenv("FAKEHOOK_LOG")
	if logPath == "" {
		os.Exit(0)
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		os.Exit(2)
	}

	defer f.Close() //nolint:errcheck // best-effort close in test helper

	// Log env vars of interest.
	for _, key := range []string{
		"SWM_HOOK", "SWM_STORY", "SWM_PROJECT_HOST",
		"SWM_PROJECT_PATH", "SWM_WORKTREE_PATH", "SWM_REPO_PATH",
	} {
		_, _ = f.WriteString(key + "=" + os.Getenv(key) + "\n") //nolint:errcheck // best-effort write in test helper
	}

	// Log stdin.
	stdin, _ := io.ReadAll(os.Stdin)                      //nolint:errcheck // best-effort read in test helper
	_, _ = f.WriteString("STDIN=" + string(stdin) + "\n") //nolint:errcheck // best-effort write in test helper
}
