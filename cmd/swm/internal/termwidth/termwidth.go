// Package termwidth detects the terminal column width.
package termwidth

import (
	"os"
	"strconv"

	"golang.org/x/sys/unix"
)

const defaultWidth = 120

// Detect returns the terminal width using the following fallback chain:
//  1. term.GetSize on /dev/tty (if the file can be opened and returns width > 0)
//  2. $COLUMNS environment variable parsed as a positive integer
//  3. defaultWidth (120)
func Detect() int {
	if w := fromTTY(); w > 0 {
		return w
	}

	return DetectFromEnv()
}

// DetectFromEnv returns the terminal width from $COLUMNS or defaultWidth.
// Useful in non-TTY contexts (scripts, CI) and for unit testing.
func DetectFromEnv() int {
	if w := fromColumns(); w > 0 {
		return w
	}

	return defaultWidth
}

func fromTTY() int {
	f, err := os.Open("/dev/tty")
	if err != nil {
		return 0
	}

	defer f.Close() //nolint:errcheck // closing a read-only device file cannot fail meaningfully

	ws, err := unix.IoctlGetWinsize(int(f.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return 0
	}

	return int(ws.Col)
}

func fromColumns() int {
	s := os.Getenv("COLUMNS")
	if s == "" {
		return 0
	}

	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0
	}

	return n
}
