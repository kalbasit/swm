package integration_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/cmd/swm/internal/cli"
	"github.com/kalbasit/swm/cmd/swm/internal/config"
	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
	"github.com/kalbasit/swm/cmd/swm/internal/core/story"
	"github.com/kalbasit/swm/cmd/swm/internal/hostsvc"
	"github.com/kalbasit/swm/cmd/swm/internal/pluginmgr"
)

const pickerPluginName = "fzf"

const (
	vcsPluginName     = "git"
	sessionPluginName = "tmux"
	testStoryName     = "feat-x"
)

// setupEnv creates an isolated environment for an integration test.
func setupEnv(t *testing.T) (*config.Config, *layout.Resolver, story.Store, *pluginmgr.Manager) {
	t.Helper()

	codeRoot := t.TempDir()
	storiesDir := filepath.Join(t.TempDir(), "stories")
	require.NoError(t, os.MkdirAll(storiesDir, 0o750))

	cfg := &config.Config{
		CodeRoot:     codeRoot,
		DefaultStory: "_default",
		Plugins: config.Plugins{
			VCS:     vcsPluginName,
			Session: sessionPluginName,
			Paths: map[string]string{
				vcsPluginName:     vcsGitBin,
				sessionPluginName: sessionTmuxBin,
			},
		},
	}

	store := story.NewJSONStore(storiesDir)
	resolver := layout.NewResolver(codeRoot)

	srv, err := hostsvc.NewServer(cfg, resolver, store)
	require.NoError(t, err)
	t.Cleanup(srv.Stop)

	mgr := pluginmgr.New(cfg, srv.SocketPath())

	t.Cleanup(func() { mgr.Close() }) //nolint:errcheck,gosec // best-effort in test cleanup

	return cfg, resolver, store, mgr
}

// initLocalRepo creates a git repo with one commit (suitable for cloning).
func initLocalRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	cmds := [][]string{
		{vcsPluginName, "-C", dir, "init"},
		{vcsPluginName, "-C", dir, "config", "user.email", "test@test.com"},
		{vcsPluginName, "-C", dir, "config", "user.name", "Test"},
		{vcsPluginName, "-C", dir, "commit", "--allow-empty", "-m", "init"},
	}
	for _, c := range cmds {
		out, err := exec.Command(c[0], c[1:]...).CombinedOutput() //nolint:gosec // trusted test commands
		require.NoError(t, err, "cmd %v: %s", c, out)
	}

	return dir
}

// fileURLtoProjectID derives the expected ProjectID from a file:// URL.
// Mirrors the logic in vcs-git's parseURL for file:// scheme.
func fileURLtoProjectID(fileURL string) *pluginv1.ProjectID {
	// "file:///tmp/foo/bar" → strip "file://" → "/tmp/foo/bar" → trim leading / → "tmp/foo/bar"
	path := strings.TrimPrefix(fileURL, "file://")
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, ".git")

	return &pluginv1.ProjectID{Host: "localhost", Segments: strings.Split(path, "/")}
}

func TestCloneAndStoryCreate(t *testing.T) {
	t.Parallel()

	cfg, resolver, store, mgr := setupEnv(t)

	srcRepo := initLocalRepo(t)
	fileURL := "file://" + srcRepo

	root := cli.NewRootCmd(cfg, mgr, store, resolver)
	root.SetArgs([]string{"clone", fileURL})
	require.NoError(t, root.Execute())

	// Verify the canonical path has a .git directory.
	pid := fileURLtoProjectID(fileURL)
	canonical := resolver.CanonicalPath(pid)
	require.DirExists(t, filepath.Join(canonical, ".git"),
		"expected .git at canonical path %s", canonical)

	// Create a story.
	root2 := cli.NewRootCmd(cfg, mgr, store, resolver)
	root2.SetArgs([]string{"story", "create", testStoryName})
	require.NoError(t, root2.Execute())

	st, err := store.Get(t.Context(), testStoryName)
	require.NoError(t, err)
	require.Equal(t, testStoryName, st.Name)
	require.Equal(t, "feat/"+testStoryName, st.BranchName)
}

func TestStoryRemove(t *testing.T) {
	t.Parallel()

	cfg, resolver, store, mgr := setupEnv(t)

	// Create story.
	_, err := store.Create(t.Context(), testStoryName, "feat/"+testStoryName)
	require.NoError(t, err)

	// Verify story exists.
	_, err = store.Get(t.Context(), testStoryName)
	require.NoError(t, err)

	// Remove story (no projects, so no VCS calls needed).
	root := cli.NewRootCmd(cfg, mgr, store, resolver)
	root.SetArgs([]string{"story", "remove", "--force", testStoryName})
	require.NoError(t, root.Execute())

	// Story should be gone.
	_, err = store.Get(t.Context(), testStoryName)
	require.Error(t, err)
}

func TestWorkspaceOpenWithPicker(t *testing.T) {
	// Override PATH so session-tmux finds faketmux as "tmux" and
	// picker-fzf finds fakefzf as "fzf".
	if _, err := os.OpenFile("/dev/tty", os.O_RDWR, 0); err != nil {
		t.Skip("no TTY available; skipping picker integration test")
	}

	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", filepath.Dir(faketmuxBin)+":"+oldPath) // faketmuxBin IS named "tmux"
	// fakefzfBin is already named "fzf" in tmpDir, which is the same dir as faketmuxBin.

	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	// Set up config with both session and picker plugins.
	codeRoot := t.TempDir()
	storiesDir := filepath.Join(t.TempDir(), "stories")
	require.NoError(t, os.MkdirAll(storiesDir, 0o750))

	cfg := &config.Config{
		CodeRoot:     codeRoot,
		DefaultStory: "_default",
		Plugins: config.Plugins{
			VCS:     vcsPluginName,
			Session: sessionPluginName,
			Picker:  pickerPluginName,
			Paths: map[string]string{
				vcsPluginName:     vcsGitBin,
				sessionPluginName: sessionTmuxBin,
				pickerPluginName:  pickerFzfBin,
			},
		},
	}

	store := story.NewJSONStore(storiesDir)
	resolver := layout.NewResolver(codeRoot)

	srv, err := hostsvc.NewServer(cfg, resolver, store)
	require.NoError(t, err)

	t.Cleanup(srv.Stop)

	mgr := pluginmgr.New(cfg, srv.SocketPath())

	t.Cleanup(func() { mgr.Close() }) //nolint:errcheck,gosec // best-effort in test cleanup

	// Clone a local repo so it appears in the candidate list.
	srcRepo := initLocalRepo(t)
	fileURL := "file://" + srcRepo

	root := cli.NewRootCmd(cfg, mgr, store, resolver)
	root.SetArgs([]string{"clone", fileURL})
	require.NoError(t, root.Execute())

	// Create a story with no projects yet (lazy attach will happen).
	_, err = store.Create(t.Context(), testStoryName, "feat/"+testStoryName)
	require.NoError(t, err)

	var buf bytes.Buffer

	root2 := cli.NewRootCmd(cfg, mgr, store, resolver)
	root2.SetArgs([]string{"workspace", "open", "--story", testStoryName})
	root2.SetOut(&buf)
	require.NoError(t, root2.Execute())

	// Verify that faketmux received a new-session call (pane group opened).
	logBytes, err := os.ReadFile(logFile) //nolint:gosec // G304: test-controlled path
	require.NoError(t, err)
	require.Contains(t, string(logBytes), "new-session", "expected new-session in faketmux log")
}

func TestWorkspaceOpen(t *testing.T) {
	// Override PATH so session-tmux finds the fake tmux binary instead of real tmux.
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", filepath.Dir(faketmuxBin)+":"+oldPath)

	cfg, resolver, store, mgr := setupEnv(t)

	// Create a story.
	_, err := store.Create(t.Context(), testStoryName, "feat/"+testStoryName)
	require.NoError(t, err)

	var buf bytes.Buffer

	root := cli.NewRootCmd(cfg, mgr, store, resolver)
	root.SetArgs([]string{"workspace", "open", "--story", testStoryName})
	root.SetOut(&buf)
	require.NoError(t, root.Execute())

	require.Contains(t, buf.String(), testStoryName)
}
