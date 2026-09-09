package config_test

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	cliconfig "github.com/kalbasit/swm/cmd/swm/internal/cli/config"

	"github.com/kalbasit/swm/cmd/swm/internal/config"
)

// keyDefaultStory is the config key these tests read; named so the assertions
// and the arguments cannot drift apart.
const keyDefaultStory = "default_story"

// captureStdout runs fn with os.Stdout replaced by a pipe and returns what was
// written to it.
//
// The other tests in this package assert content by calling cmd.SetOut, and
// that is exactly why they could not catch this bug: cobra's Print/Println
// write to OutOrStderr(), which returns the *out* writer once SetOut has been
// called. With SetOut, stdout and stderr become the same buffer and the
// distinction the caller depends on disappears. The only way to see which
// stream a value actually lands on is to leave the command's writers alone and
// capture the real file.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	require.NoError(t, err)

	saved := os.Stdout
	os.Stdout = w

	done := make(chan string, 1)

	go func() {
		b, readErr := io.ReadAll(r)
		if readErr != nil {
			done <- ""

			return
		}

		done <- string(b)
	}()

	// Deferred, because fn asserts: a failing require calls FailNow, which
	// unwinds through runtime.Goexit. That runs deferred functions and skips
	// everything after the call -- so without this, one failing test would
	// leave os.Stdout pointed at a pipe nobody reads for the rest of the
	// process, and every later test writing to stdout would fill it and block.
	// Closing w here also releases the reader goroutine on that path.
	defer func() {
		os.Stdout = saved
		//nolint:errcheck // already closed on the success path; the error says exactly that
		_ = w.Close()
		//nolint:errcheck // nothing can act on a failure to close a test pipe
		_ = r.Close()
	}()

	fn()

	// Closed before reading, or ReadAll never returns.
	require.NoError(t, w.Close())

	return <-done
}

// TestGetWritesToStdout is the property `swm config get` exists for:
// `root=$(swm config get code_root)` must capture the value.
//
// It did not. The value went to stderr, so the assignment came back empty, and
// a steward host agent reported the code root as missing on a machine where it
// was configured.
//
//nolint:paralleltest // swaps os.Stdout, which is process-global
func TestGetWritesToStdout(t *testing.T) {
	cfg := config.Defaults()

	got := captureStdout(t, func() {
		cmd := cliconfig.NewGetCmd(cfg)
		cmd.SetArgs([]string{keyDefaultStory})
		require.NoError(t, cmd.Execute())
	})

	require.Equal(t, "_default\n", got,
		"the value must reach stdout, or $(swm config get ...) captures nothing")
}

// TestListWritesToStdout covers the same bug in the sibling command, which a
// caller pipes into grep or awk for the same reason.
//
//nolint:paralleltest // swaps os.Stdout, which is process-global
func TestListWritesToStdout(t *testing.T) {
	cfg := config.Defaults()

	got := captureStdout(t, func() {
		cmd := cliconfig.NewListCmd("", cfg)
		cmd.SetArgs([]string{testFlagAll})
		require.NoError(t, cmd.Execute())
	})

	require.Contains(t, got, keyDefaultStory+" = _default",
		"the listing must reach stdout, or piping it yields nothing")
}
