//go:build !windows

package termwidth

import (
	"os"

	"golang.org/x/sys/unix"
)

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
