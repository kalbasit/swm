package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	testDefaultStory  = "_default"
	cmdCreate         = "create"
	cmdRemove         = "remove"
	cmdGroupStory     = "story"
	cmdGroupWorkspace = "workspace"
	cmdOpen           = "open"
	cmdClone          = "clone"
	flagStory         = "--story"
	flagForce         = "--force"
)

// setupEnv creates an isolated environment for an integration test.
func setupEnv(t *testing.T) (*config.Config, *layout.Resolver, story.Store, *pluginmgr.Manager) {
	t.Helper()

	codeRoot := t.TempDir()
	storiesDir := filepath.Join(t.TempDir(), "stories")
	require.NoError(t, os.MkdirAll(storiesDir, 0o750))

	cfg := &config.Config{
		CodeRoot:     codeRoot,
		DefaultStory: testDefaultStory,
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
	resolver := layout.NewResolver(codeRoot, testDefaultStory)

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

	root := cli.NewRootCmd("", cfg, mgr, store, resolver)
	root.SetArgs([]string{cmdClone, fileURL})
	require.NoError(t, root.Execute())

	// Verify the canonical path has a .git directory.
	pid := fileURLtoProjectID(fileURL)
	canonical := resolver.CanonicalPath(pid)
	require.DirExists(t, filepath.Join(canonical, ".git"),
		"expected .git at canonical path %s", canonical)

	// Create a story.
	root2 := cli.NewRootCmd("", cfg, mgr, store, resolver)
	root2.SetArgs([]string{cmdGroupStory, cmdCreate, testStoryName})
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
	root := cli.NewRootCmd("", cfg, mgr, store, resolver)
	root.SetArgs([]string{cmdGroupStory, cmdRemove, flagForce, testStoryName})
	require.NoError(t, root.Execute())

	// Story should be gone.
	_, err = store.Get(t.Context(), testStoryName)
	require.Error(t, err)
}

func TestStoryRemove_NoArg_SWMStorySet(t *testing.T) {
	// No t.Parallel(): t.Setenv is not allowed in parallel tests.
	t.Setenv("SWM_STORY", testStoryName)

	cfg, resolver, store, mgr := setupEnv(t)

	_, err := store.Create(t.Context(), testStoryName, "feat/"+testStoryName)
	require.NoError(t, err)

	root := cli.NewRootCmd("", cfg, mgr, store, resolver)
	root.SetArgs([]string{cmdGroupStory, cmdRemove, flagForce}) // no name arg
	require.NoError(t, root.Execute())

	_, err = store.Get(t.Context(), testStoryName)
	require.Error(t, err, "story should be deleted when $SWM_STORY is set and no arg provided")
}

func TestStoryRemove_NoArg_SWMStoryUnset_ReturnsError(t *testing.T) {
	// No t.Parallel(): t.Setenv is not allowed in parallel tests.
	t.Setenv("SWM_STORY", "")

	cfg, resolver, store, mgr := setupEnv(t)

	_, err := store.Create(t.Context(), testStoryName, "feat/"+testStoryName)
	require.NoError(t, err)

	root := cli.NewRootCmd("", cfg, mgr, store, resolver)
	root.SetArgs([]string{cmdGroupStory, cmdRemove, flagForce}) // no name, no env
	require.Error(t, root.Execute(), "should error when neither arg nor $SWM_STORY is set")

	_, err = store.Get(t.Context(), testStoryName)
	require.NoError(t, err, "story should NOT be deleted when command errors")
}

func TestStoryRemove_ExplicitArg_OverridesSWMStory(t *testing.T) {
	// No t.Parallel(): t.Setenv is not allowed in parallel tests.
	t.Setenv("SWM_STORY", "other-story")

	cfg, resolver, store, mgr := setupEnv(t)

	_, err := store.Create(t.Context(), testStoryName, "feat/"+testStoryName)
	require.NoError(t, err)

	root := cli.NewRootCmd("", cfg, mgr, store, resolver)
	root.SetArgs([]string{cmdGroupStory, cmdRemove, flagForce, testStoryName}) // explicit arg
	require.NoError(t, root.Execute())

	_, err = store.Get(t.Context(), testStoryName)
	require.Error(t, err, "explicitly named story should be deleted, not the env-var story")
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
		DefaultStory: testDefaultStory,
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
	resolver := layout.NewResolver(codeRoot, testDefaultStory)

	srv, err := hostsvc.NewServer(cfg, resolver, store)
	require.NoError(t, err)

	t.Cleanup(srv.Stop)

	mgr := pluginmgr.New(cfg, srv.SocketPath())

	t.Cleanup(func() { mgr.Close() }) //nolint:errcheck,gosec // best-effort in test cleanup

	// Clone a local repo so it appears in the candidate list.
	srcRepo := initLocalRepo(t)
	fileURL := "file://" + srcRepo

	root := cli.NewRootCmd("", cfg, mgr, store, resolver)
	root.SetArgs([]string{cmdClone, fileURL})
	require.NoError(t, root.Execute())

	// Create a story with no projects yet (lazy attach will happen).
	_, err = store.Create(t.Context(), testStoryName, "feat/"+testStoryName)
	require.NoError(t, err)

	var buf bytes.Buffer

	root2 := cli.NewRootCmd("", cfg, mgr, store, resolver)
	root2.SetArgs([]string{cmdGroupWorkspace, cmdOpen, testStoryName})
	root2.SetOut(&buf)
	require.NoError(t, root2.Execute())

	// Verify that faketmux received a new-session call (pane group opened).
	logBytes, err := os.ReadFile(logFile) //nolint:gosec // G304: test-controlled path
	require.NoError(t, err)
	require.Contains(t, string(logBytes), "new-session", "expected new-session in faketmux log")
}

// setupPickerEnv creates an isolated environment configured with vcs, session,
// and picker plugins using the fake binaries. It also verifies that /dev/tty
// is available and skips the test if not.
func setupPickerEnv(t *testing.T) (*config.Config, *layout.Resolver, story.Store, *pluginmgr.Manager, string) {
	t.Helper()

	if _, err := os.OpenFile("/dev/tty", os.O_RDWR, 0); err != nil {
		t.Skip("no TTY available; skipping story picker integration test")
	}

	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", filepath.Dir(faketmuxBin)+":"+oldPath)

	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	codeRoot := t.TempDir()
	storiesDir := filepath.Join(t.TempDir(), "stories")
	require.NoError(t, os.MkdirAll(storiesDir, 0o750))

	cfg := &config.Config{
		CodeRoot:     codeRoot,
		DefaultStory: testDefaultStory,
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

	st := story.NewJSONStore(storiesDir)
	resolver := layout.NewResolver(codeRoot, testDefaultStory)

	srv, err := hostsvc.NewServer(cfg, resolver, st)
	require.NoError(t, err)

	t.Cleanup(srv.Stop)

	mgr := pluginmgr.New(cfg, srv.SocketPath())

	t.Cleanup(func() { mgr.Close() }) //nolint:errcheck,gosec // best-effort in test cleanup

	return cfg, resolver, st, mgr, logFile
}

// TestWorkspaceOpenWithStoryPicker verifies that when workspace open is called
// with no story argument or $SWM_STORY, the story picker is shown and the
// selected story's workspace is opened.
func TestWorkspaceOpenWithStoryPicker(t *testing.T) { //nolint:paralleltest // uses t.Setenv via setupPickerEnv
	cfg, resolver, store, mgr, logFile := setupPickerEnv(t)

	// Create the feat-x story (the only non-default story in the picker;
	// fakefzf will select it as the first entry).
	_, err := store.Create(t.Context(), testStoryName, "feat/"+testStoryName)
	require.NoError(t, err)

	// Clone a local repo so it appears in the project candidate list.
	srcRepo := initLocalRepo(t)
	fileURL := "file://" + srcRepo

	root := cli.NewRootCmd("", cfg, mgr, store, resolver)
	root.SetArgs([]string{cmdClone, fileURL})
	require.NoError(t, root.Execute())

	// Run workspace open with NO story argument — story picker must be shown.
	var buf bytes.Buffer

	root2 := cli.NewRootCmd("", cfg, mgr, store, resolver)
	root2.SetArgs([]string{cmdGroupWorkspace, cmdOpen}) // no story name
	root2.SetOut(&buf)
	require.NoError(t, root2.Execute())

	// faketmux received a new-session call (workspace was opened).
	logBytes, err := os.ReadFile(logFile) //nolint:gosec // G304: test-controlled path
	require.NoError(t, err)
	require.Contains(t, string(logBytes), "new-session", "expected new-session in faketmux log")

	// The project was attached to the correct story (feat-x, not any other).
	st, err := store.Get(t.Context(), testStoryName)
	require.NoError(t, err)
	require.NotEmpty(t, st.Projects, "expected at least one project attached to the selected story")
}

// TestWorkspaceOpenWithSWMStorySkipsStoryPicker verifies that when $SWM_STORY
// is set, the story picker is not shown and the specified story is opened.
// To detect picker invocation, a second story (feat-z, more recent) is created;
// if the story picker ran, fakefzf would select feat-z instead of feat-x.
func TestWorkspaceOpenWithSWMStorySkipsStoryPicker(t *testing.T) {
	cfg, resolver, store, mgr, logFile := setupPickerEnv(t)

	// Create feat-x first (older).
	_, err := store.Create(t.Context(), testStoryName, "feat/"+testStoryName)
	require.NoError(t, err)

	// Create feat-z second (newer — fakefzf would select this if story picker ran).
	_, err = store.Create(t.Context(), "feat-z", "feat/feat-z")
	require.NoError(t, err)

	// Clone a repo so project picker has a candidate.
	srcRepo := initLocalRepo(t)
	fileURL := "file://" + srcRepo

	root := cli.NewRootCmd("", cfg, mgr, store, resolver)
	root.SetArgs([]string{cmdClone, fileURL})
	require.NoError(t, root.Execute())

	// Set SWM_STORY so the story picker must be bypassed.
	t.Setenv("SWM_STORY", testStoryName)

	var buf bytes.Buffer

	root2 := cli.NewRootCmd("", cfg, mgr, store, resolver)
	root2.SetArgs([]string{cmdGroupWorkspace, cmdOpen}) // no positional arg
	root2.SetOut(&buf)
	require.NoError(t, root2.Execute())

	// faketmux received new-session (workspace opened).
	logBytes, err := os.ReadFile(logFile) //nolint:gosec // G304: test-controlled path
	require.NoError(t, err)
	require.Contains(t, string(logBytes), "new-session")

	// Project was attached to feat-x (not feat-z), confirming story picker was skipped.
	stX, err := store.Get(t.Context(), testStoryName)
	require.NoError(t, err)
	require.NotEmpty(t, stX.Projects, "project must be attached to feat-x (the SWM_STORY value)")

	stZ, err := store.Get(t.Context(), "feat-z")
	require.NoError(t, err)
	require.Empty(t, stZ.Projects, "feat-z must have no projects (story picker must have been skipped)")
}

// TestWorkspaceOpenWithPositionalArgSkipsStoryPicker verifies that a positional
// story argument bypasses the story picker, analogous to the $SWM_STORY test.
func TestWorkspaceOpenWithPositionalArgSkipsStoryPicker(t *testing.T) { //nolint:paralleltest // t.Setenv in helper
	cfg, resolver, store, mgr, logFile := setupPickerEnv(t)

	_, err := store.Create(t.Context(), testStoryName, "feat/"+testStoryName)
	require.NoError(t, err)

	// Newer story — would be selected by fakefzf if story picker ran.
	_, err = store.Create(t.Context(), "feat-z", "feat/feat-z")
	require.NoError(t, err)

	srcRepo := initLocalRepo(t)
	fileURL := "file://" + srcRepo

	root := cli.NewRootCmd("", cfg, mgr, store, resolver)
	root.SetArgs([]string{cmdClone, fileURL})
	require.NoError(t, root.Execute())

	var buf bytes.Buffer

	root2 := cli.NewRootCmd("", cfg, mgr, store, resolver)
	root2.SetArgs([]string{cmdGroupWorkspace, cmdOpen, testStoryName}) // positional arg provided
	root2.SetOut(&buf)
	require.NoError(t, root2.Execute())

	logBytes, err := os.ReadFile(logFile) //nolint:gosec // G304: test-controlled path
	require.NoError(t, err)
	require.Contains(t, string(logBytes), "new-session")

	stX, err := store.Get(t.Context(), testStoryName)
	require.NoError(t, err)
	require.NotEmpty(t, stX.Projects, "project must be attached to feat-x (the positional arg)")

	stZ, err := store.Get(t.Context(), "feat-z")
	require.NoError(t, err)
	require.Empty(t, stZ.Projects, "feat-z must have no projects (story picker must have been skipped)")
}

// TestWorkspaceOpenStoryPickerRespectsTerminalWidth verifies that the story
// picker still works correctly when COLUMNS is set to a narrow width (80).
// Correct truncation is verified in unit tests; this confirms no runtime error.
func TestWorkspaceOpenStoryPickerRespectsTerminalWidth(t *testing.T) {
	cfg, resolver, store, mgr, logFile := setupPickerEnv(t)

	t.Setenv("COLUMNS", "80")

	_, err := store.Create(t.Context(), testStoryName, "feat/"+testStoryName)
	require.NoError(t, err)

	srcRepo := initLocalRepo(t)
	fileURL := "file://" + srcRepo

	root := cli.NewRootCmd("", cfg, mgr, store, resolver)
	root.SetArgs([]string{cmdClone, fileURL})
	require.NoError(t, root.Execute())

	var buf bytes.Buffer

	root2 := cli.NewRootCmd("", cfg, mgr, store, resolver)
	root2.SetArgs([]string{cmdGroupWorkspace, cmdOpen}) // no story name — triggers story picker
	root2.SetOut(&buf)
	require.NoError(t, root2.Execute())

	logBytes, err := os.ReadFile(logFile) //nolint:gosec // G304: test-controlled path
	require.NoError(t, err)
	require.Contains(t, string(logBytes), "new-session")
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

	root := cli.NewRootCmd("", cfg, mgr, store, resolver)
	root.SetArgs([]string{cmdGroupWorkspace, cmdOpen, testStoryName})
	root.SetOut(&buf)
	require.NoError(t, root.Execute())

	require.Contains(t, buf.String(), testStoryName)
}

const forgePluginName = "github"

// fakePR builds the minimal GitHub API JSON for a pull request.
func fakePR(number int, title, htmlURL, head, base string) map[string]any {
	return map[string]any{
		"number":   number,
		"title":    title,
		"html_url": htmlURL,
		"state":    "open",
		"draft":    false,
		"body":     "",
		"head":     map[string]any{"ref": head, "sha": "abc123"},
		"base":     map[string]any{"ref": base, "sha": "def456"},
	}
}

func TestPRListAndCreate(t *testing.T) {
	// GitHub API mock server.
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/kalbasit/swm/pulls", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.Method {
		case http.MethodGet:
			prs := []map[string]any{
				fakePR(42, "Test PR", "https://github.com/kalbasit/swm/pull/42", "feat/test", "main"),
			}

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(prs) //nolint:errcheck // test mock, response write failure is non-critical
		case http.MethodPost:
			pr := fakePR(43, "New PR", "https://github.com/kalbasit/swm/pull/43", "feat/new", "main")

			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(pr) //nolint:errcheck // test mock, response write failure is non-critical
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	apiServer := httptest.NewServer(mux)
	t.Cleanup(apiServer.Close)

	// Tell the forge-github subprocess to use the test server instead of api.github.com.
	t.Setenv("FORGE_GITHUB_API_URL", apiServer.URL+"/")

	// Token file the forge-github plugin will read.
	tokenFile := filepath.Join(t.TempDir(), "github_token")
	require.NoError(t, os.WriteFile(tokenFile, []byte("fake-token"), 0o600))

	codeRoot := t.TempDir()
	storiesDir := filepath.Join(t.TempDir(), "stories")
	require.NoError(t, os.MkdirAll(storiesDir, 0o750))

	cfg := &config.Config{
		CodeRoot:     codeRoot,
		DefaultStory: testDefaultStory,
		Plugins: config.Plugins{
			VCS:    vcsPluginName,
			Forges: []string{forgePluginName},
			Paths: map[string]string{
				vcsPluginName:   vcsGitBin,
				forgePluginName: forgeGithubBin,
			},
			Config: map[string]map[string]any{
				"forge-github": {"token_path": tokenFile},
			},
		},
	}

	st := story.NewJSONStore(storiesDir)
	resolver := layout.NewResolver(codeRoot, testDefaultStory)

	srv, err := hostsvc.NewServer(cfg, resolver, st)
	require.NoError(t, err)
	t.Cleanup(srv.Stop)

	mgr := pluginmgr.New(cfg, srv.SocketPath())

	t.Cleanup(func() { mgr.Close() }) //nolint:errcheck,gosec // best-effort cleanup

	// Create a story with a github.com project attached so pr list has something to query.
	s, err := st.Create(t.Context(), "feat-pr", "feat/feat-pr")
	require.NoError(t, err)

	s.Projects = []story.Project{{Host: "github.com", Segments: []string{"kalbasit", "swm"}}}
	require.NoError(t, st.Update(t.Context(), s))

	// --- pr list ---
	var listBuf bytes.Buffer

	root := cli.NewRootCmd("", cfg, mgr, st, resolver)
	root.SetArgs([]string{"pr", "list", flagStory, "feat-pr"})
	root.SetOut(&listBuf)
	require.NoError(t, root.Execute())

	listOut := listBuf.String()
	require.Contains(t, listOut, "#42")
	require.Contains(t, listOut, "Test PR")
	require.Contains(t, listOut, "https://github.com/kalbasit/swm/pull/42")

	// --- pr create (from inside a project directory that resolves to github.com/kalbasit/swm) ---
	projectDir := filepath.Join(codeRoot, "repositories", "github.com", "kalbasit", "swm")
	require.NoError(t, os.MkdirAll(projectDir, 0o750))
	t.Chdir(projectDir)

	var createBuf bytes.Buffer

	root2 := cli.NewRootCmd("", cfg, mgr, st, resolver)
	root2.SetArgs([]string{"pr", "create", "--title", "New PR", "--head", "feat/new"})
	root2.SetOut(&createBuf)
	require.NoError(t, root2.Execute())

	require.Contains(t, createBuf.String(), "https://github.com/kalbasit/swm/pull/43")
}

func TestHooksRunOnStoryCreate(t *testing.T) {
	hooksConfigHome := t.TempDir()

	// Create the global pre-story-create hook directory.
	hookDir := filepath.Join(hooksConfigHome, "swm", "hooks", "pre-story-create.d")
	require.NoError(t, os.MkdirAll(hookDir, 0o750))

	// Write a small shell script that creates a sentinel file.
	sentinelFile := filepath.Join(t.TempDir(), "hook_ran")
	t.Setenv("HOOK_SENTINEL_FILE", sentinelFile)

	hookScript := filepath.Join(hookDir, "01-sentinel.sh")
	//nolint:gosec // G306: hook script must be executable
	require.NoError(t, os.WriteFile(hookScript,
		fmt.Appendf(nil, "#!/bin/sh\ntouch %q\n", sentinelFile), 0o750))

	cfg, resolver, st, mgr := setupEnv(t)
	cfg.HooksConfigHome = hooksConfigHome

	root := cli.NewRootCmd("", cfg, mgr, st, resolver)
	root.SetArgs([]string{cmdGroupStory, cmdCreate, testStoryName})
	require.NoError(t, root.Execute())

	require.FileExists(t, sentinelFile, "expected pre-story-create hook to create sentinel file")
}
