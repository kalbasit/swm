// faketmux is a fake tmux binary used in unit tests.
// It records invocations and simulates tmux behaviour via socket files.
package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	args := os.Args[1:]

	// Record invocation for test assertions.
	if logFile := os.Getenv("FAKETMUX_LOG"); logFile != "" {
		f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err == nil {
			fmt.Fprintln(f, strings.Join(args, " "))
			f.Close() //nolint:errcheck // best-effort log
		}
	}

	// Record environment for env-isolation test assertions.
	if envFile := os.Getenv("FAKETMUX_ENV_LOG"); envFile != "" {
		f, err := os.OpenFile(envFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err == nil {
			for _, e := range os.Environ() {
				fmt.Fprintln(f, e)
			}
			f.Close() //nolint:errcheck // best-effort log
		}
	}

	// Parse -S <socket> and the subcommand.
	socket, cmd := parseArgs(args)

	switch cmd {
	case "new-session":
		if socket != "" {
			os.WriteFile(socket, nil, 0o600) //nolint:errcheck // fake socket creation
		}
	case "kill-server":
		if socket != "" {
			os.Remove(socket) //nolint:errcheck // fake socket removal
		}
	case "list-sessions":
		if socket != "" {
			if _, err := os.Stat(socket); err != nil {
				os.Exit(1)
			}
		}
	case "has-session":
		// Default: session not found so the caller creates it.
		// Set FAKETMUX_HAS_SESSION=0 to simulate an existing session.
		if os.Getenv("FAKETMUX_HAS_SESSION") == "0" {
			os.Exit(0)
		}
		os.Exit(1)
	case "kill-pane":
		if os.Getenv("FAKETMUX_KILL_PANE_FAIL") == "1" {
			fmt.Fprintln(os.Stderr, "no such pane")
			os.Exit(1)
		}
	case "display-message":
		name := os.Getenv("FAKETMUX_SESSION")
		if name == "" {
			name = "test-session"
		}
		fmt.Println(name)
	case "split-window":
		// Return a fake pane ID so layout.Apply can reference the new pane.
		fmt.Println("%1")
	}
}

func parseArgs(args []string) (socket, cmd string) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-S":
			if i+1 < len(args) {
				socket = args[i+1]
				i++
			}
		default:
			if !strings.HasPrefix(args[i], "-") && cmd == "" && args[i] != socket {
				cmd = args[i]
			}
		}
	}
	return
}
