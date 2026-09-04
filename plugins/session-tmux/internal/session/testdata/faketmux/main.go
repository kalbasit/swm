// faketmux is a fake tmux binary used in unit tests.
// It records invocations and simulates tmux behaviour via socket files.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strconv"
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
			recordSession(socket, flagValue(args, "-s"))
		}
	case "kill-server":
		if socket != "" {
			os.Remove(socket)                //nolint:errcheck // fake socket removal
			os.Remove(sessionsFile(socket))  //nolint:errcheck // fake session list removal
			os.Remove(panesFile(socket))     //nolint:errcheck // fake pane list removal
			os.Remove(paneCountFile(socket)) //nolint:errcheck // fake pane counter removal
		}
	case "list-sessions":
		if socket != "" {
			if _, err := os.Stat(socket); err != nil {
				os.Exit(1)
			}
		}
	case "has-session":
		// FAKETMUX_HAS_SESSION is an explicit override for tests that do not
		// care which sessions exist: "0" forces found, anything else non-empty
		// forces not-found. When unset, answer from the recorded session list
		// using tmux's own target resolution.
		switch os.Getenv("FAKETMUX_HAS_SESSION") {
		case "0":
			os.Exit(0)
		case "":
			if resolveTarget(flagValue(args, "-t"), readSessions(socket)) {
				os.Exit(0)
			}
			os.Exit(1)
		default:
			os.Exit(1)
		}
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
	case "new-window":
		// FAKETMUX_NEW_WINDOW_FAIL simulates tmux refusing to create a window
		// because the target pane group does not exist.
		if os.Getenv("FAKETMUX_NEW_WINDOW_FAIL") == "1" {
			fmt.Fprintln(os.Stderr, "can't find session")
			os.Exit(1)
		}

		paneID := nextPaneID(socket)
		recordPane(socket, paneID, strings.TrimPrefix(flagValue(args, "-t"), "="), flagValue(args, "-c"))
		fmt.Println(paneID)
	case "list-panes":
		// The rows are pre-formatted by whoever recorded them, so the fake does
		// not have to implement tmux's -F format expansion. Tests that care
		// about specific field values seed the file directly.
		if socket != "" {
			if b, err := os.ReadFile(panesFile(socket)); err == nil {
				os.Stdout.Write(b) //nolint:errcheck // best-effort fake output
			}
		}
	}
}

// panesFile is the path where the fake records the panes that exist on a given
// socket, one tab-separated row per pane.
func panesFile(socket string) string { return socket + ".panes" }

// paneCountFile is the path where the fake keeps its per-socket pane ID counter.
func paneCountFile(socket string) string { return socket + ".panecount" }

// nextPaneID mints a tmux-style pane ID ("%N") for socket, incrementing a
// per-socket counter so that two panes never share an ID.
func nextPaneID(socket string) string {
	if socket == "" {
		return "%0"
	}

	counter := paneCountFile(socket)

	n := 0
	if b, err := os.ReadFile(counter); err == nil {
		if parsed, err := strconv.Atoi(strings.TrimSpace(string(b))); err == nil {
			n = parsed
		}
	}

	n++

	os.WriteFile(counter, []byte(strconv.Itoa(n)), 0o600) //nolint:errcheck // best-effort fake state

	return "%" + strconv.Itoa(n)
}

// recordPane appends a pane row for socket in the same field order production
// code asks tmux for: pane ID, session, title, current command, current path,
// session_attached, window_active, pane_active.
//
// A pane created by the fake is reported as unattached and inactive, so tests
// that want the focused case seed their own rows rather than getting it by
// accident.
func recordPane(socket, paneID, session, path string) {
	if socket == "" {
		return
	}

	f, err := os.OpenFile(panesFile(socket), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close() //nolint:errcheck // best-effort record

	fmt.Fprintf(f, "%s\t%s\t%s\t%s\t%s\t0\t0\t0\n", paneID, session, "fake", "fake", path)
}

// resolveTarget reports whether target selects one of sessions.
//
// It deliberately mirrors the target-session resolution rules from tmux(1):
// a target prefixed with "=" matches only an exact name, otherwise tmux tries
// an exact match, then a name prefix, then an fnmatch(3) pattern.
//
// There is deliberately no substring pass: tmux does not substring-match a
// target-session. Verified against tmux 3.6a — for a session "abc-name-xyz",
// the targets "abc" (prefix), "abc*" and "*name*" (fnmatch) all resolve to it,
// but the bare substring "name" does not.
//
// Modelling that order is the whole point of this fake — without it a test
// cannot express "a session with a similar name exists", which is exactly the
// state that makes an unescaped target resolve to the wrong session.
func resolveTarget(target string, sessions []string) bool {
	if target == "" {
		return len(sessions) > 0
	}

	// Strip a window/pane suffix; only the session component is resolved here.
	if i := strings.IndexAny(target, ":"); i >= 0 {
		target = target[:i]
	}

	if exact, ok := strings.CutPrefix(target, "="); ok {
		for _, s := range sessions {
			if s == exact {
				return true
			}
		}

		return false
	}

	for _, s := range sessions {
		if s == target {
			return true
		}
	}

	for _, s := range sessions {
		if strings.HasPrefix(s, target) {
			return true
		}
	}

	for _, s := range sessions {
		if ok, err := path.Match(target, s); err == nil && ok {
			return true
		}
	}

	return false
}

// sessionsFile is the path where the fake records the names of sessions
// created on a given socket.
func sessionsFile(socket string) string { return socket + ".sessions" }

// recordSession appends name to the socket's session list.
func recordSession(socket, name string) {
	if socket == "" || name == "" {
		return
	}

	f, err := os.OpenFile(sessionsFile(socket), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close() //nolint:errcheck // best-effort record

	fmt.Fprintln(f, name)
}

// readSessions returns the names of sessions recorded for socket.
func readSessions(socket string) []string {
	if socket == "" {
		return nil
	}

	f, err := os.Open(sessionsFile(socket))
	if err != nil {
		return nil
	}
	defer f.Close() //nolint:errcheck // read-only

	var names []string

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if n := strings.TrimSpace(sc.Text()); n != "" {
			names = append(names, n)
		}
	}

	return names
}

// flagValue returns the argument following flag, or "" when absent.
func flagValue(args []string, flag string) string {
	for i := range args {
		if args[i] == flag && i+1 < len(args) {
			return args[i+1]
		}
	}

	return ""
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
