package story_test

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"

	"github.com/kalbasit/swm/cmd/swm/internal/cli/story"
)

// captureStdout runs fn with os.Stdout replaced by a pipe and returns what was
// written to it.
//
// The other tests in this package assert content by calling cmd.SetOut, and
// that is why they could not catch this: cobra's Print family writes to
// OutOrStderr(), which returns the *out* writer once SetOut has been called.
// With SetOut the two streams become one buffer and the distinction a caller
// depends on disappears inside the test meant to check it. The only vantage
// point where it exists is the process's own stdout.
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

	// Deferred: fn asserts, and a failing require unwinds through
	// runtime.Goexit, which runs defers and skips everything after the call.
	// Without this, one failure would leave os.Stdout pointed at a pipe nobody
	// reads for the rest of the process.
	defer func() {
		os.Stdout = saved
		//nolint:errcheck // already closed on the success path; the error says so
		_ = w.Close()
		//nolint:errcheck // nothing can act on a failure to close a test pipe
		_ = r.Close()
	}()

	fn()

	require.NoError(t, w.Close())

	return <-done
}

// TestListWritesNamesToStdout is the property the command exists for:
// `swm story list | grep -qx feat-x` must find a story that exists.
//
//nolint:paralleltest // swaps os.Stdout, which is process-global
func TestListWritesNamesToStdout(t *testing.T) {
	store := &stubStore{listStories: []*coreStory.Story{
		{Name: testStoryName},
		{Name: testBugName},
	}}

	got := captureStdout(t, func() {
		cmd := story.NewListCmd(store, "_default")
		cmd.SetArgs(nil)
		require.NoError(t, cmd.Execute())
	})

	for _, want := range []string{testStoryName, testBugName} {
		require.Contains(t, got, want,
			"%q missing from stdout; a caller capturing this reads an empty list", want)
	}
}
