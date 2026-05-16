package hookexec_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kalbasit/swm/cmd/swm/internal/hookexec"
)

const testStoryName = "feat-x"

var fakehookBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "hookexec-fakehook-*")
	if err != nil {
		panic("create temp dir: " + err.Error())
	}

	defer os.RemoveAll(dir) //nolint:errcheck // best-effort cleanup in TestMain

	fakehookBin = filepath.Join(dir, "fakehook")

	_, thisFile, _, _ := runtime.Caller(0)
	fakehookSrc := filepath.Join(filepath.Dir(thisFile), "testdata", "fakehook")

	buildCmd := exec.Command("go", "build", "-o", fakehookBin, fakehookSrc) //nolint:gosec // test build from trusted path

	out, err := buildCmd.CombinedOutput()
	if err != nil {
		panic("build fakehook: " + string(out))
	}

	os.Exit(m.Run())
}

// writeScript writes an executable shell script to path.
func writeScript(t *testing.T, path, body string) {
	t.Helper()

	//nolint:gosec // G306: hook scripts must be executable
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o750))
}

// installScript installs a shell script as an executable hook in tier/eventDir/name.
func installScript(t *testing.T, tierDir, event, name, body string) {
	t.Helper()

	hookDir := filepath.Join(tierDir, event+".d")
	require.NoError(t, os.MkdirAll(hookDir, 0o750))

	writeScript(t, filepath.Join(hookDir, name), body)
}

// installFakehook installs the compiled fakehook binary as a hook.
func installFakehook(t *testing.T, tierDir, event, name string) {
	t.Helper()

	hookDir := filepath.Join(tierDir, event+".d")
	require.NoError(t, os.MkdirAll(hookDir, 0o750))

	data, err := os.ReadFile(fakehookBin) //nolint:gosec // G304: reading trusted test binary
	require.NoError(t, err)

	//nolint:gosec // G306: hook binary must be executable
	require.NoError(t, os.WriteFile(filepath.Join(hookDir, name), data, 0o750))
}

func TestRun_NoHooksExist(t *testing.T) {
	t.Parallel()

	// No hooks exist — should succeed silently.
	cfg := hookexec.RunConfig{
		Event:      "pre-story-create",
		CodeRoot:   t.TempDir(),
		StoryName:  testStoryName,
		ConfigHome: t.TempDir(),
	}

	require.NoError(t, hookexec.Run(context.Background(), cfg))
}

func TestRun_LexicalOrder(t *testing.T) {
	t.Parallel()

	configHome := t.TempDir()

	sentinelDir := t.TempDir()
	firstFile := filepath.Join(sentinelDir, "00-first.ran")
	secondFile := filepath.Join(sentinelDir, "10-second.ran")

	globalDir := filepath.Join(configHome, "swm", "hooks")
	installScript(t, globalDir, "post-story-create", "00-first",
		"touch "+firstFile+"\n")
	installScript(t, globalDir, "post-story-create", "10-second",
		"touch "+secondFile+"\n")

	cfg := hookexec.RunConfig{
		Event:      "post-story-create",
		CodeRoot:   t.TempDir(),
		StoryName:  testStoryName,
		ConfigHome: configHome,
	}

	require.NoError(t, hookexec.Run(context.Background(), cfg))
	require.FileExists(t, firstFile, "00-first hook should have run")
	require.FileExists(t, secondFile, "10-second hook should have run")
}

func TestRun_PreHookAborts(t *testing.T) {
	configHome := t.TempDir()

	globalDir := filepath.Join(configHome, "swm", "hooks")
	installFakehook(t, globalDir, "pre-worktree-create", "00-fail")

	logFile := filepath.Join(t.TempDir(), "fakehook.log")

	cfg := hookexec.RunConfig{
		Event:      "pre-worktree-create",
		CodeRoot:   t.TempDir(),
		StoryName:  testStoryName,
		ConfigHome: configHome,
		// Ask fakehook to exit non-zero via env (inherited from test process).
		// We set FAKEHOOK_EXIT via the hook's env by embedding it in the binary path
		// lookup — instead, use a wrapper script that sets the env and execs fakehook.
	}

	// Temporarily set FAKEHOOK_EXIT=1 for this test only.
	t.Setenv("FAKEHOOK_EXIT", "1")

	err := hookexec.Run(context.Background(), cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "pre-hook")

	_ = logFile
}

func TestRun_PostHookFailsButContinues(t *testing.T) {
	t.Parallel()

	configHome := t.TempDir()

	sentinelDir := t.TempDir()
	sentinelFile := filepath.Join(sentinelDir, "10-succeed.ran")

	globalDir := filepath.Join(configHome, "swm", "hooks")
	// 00-fail exits 1; 10-succeed should still run.
	installScript(t, globalDir, "post-story-create", "00-fail", "exit 1\n")
	installScript(t, globalDir, "post-story-create", "10-succeed",
		"touch "+sentinelFile+"\n")

	cfg := hookexec.RunConfig{
		Event:      "post-story-create",
		CodeRoot:   t.TempDir(),
		StoryName:  testStoryName,
		ConfigHome: configHome,
	}

	require.NoError(t, hookexec.Run(context.Background(), cfg))
	require.FileExists(t, sentinelFile, "10-succeed hook should have run despite 00-fail failing")
}

func TestRun_EnvVarsSet(t *testing.T) {
	t.Parallel()

	configHome := t.TempDir()

	logFile := filepath.Join(t.TempDir(), "env.log")
	// Script dumps relevant env vars into the log file.
	script := `printf 'SWM_HOOK=%s\nSWM_STORY=%s\nSWM_PROJECT_HOST=%s\nSWM_PROJECT_PATH=%s\n' \
		"$SWM_HOOK" "$SWM_STORY" "$SWM_PROJECT_HOST" "$SWM_PROJECT_PATH" > ` + logFile + "\n"

	globalDir := filepath.Join(configHome, "swm", "hooks")
	installScript(t, globalDir, "pre-worktree-create", "00-check", script)

	cfg := hookexec.RunConfig{
		Event:        "pre-worktree-create",
		CodeRoot:     t.TempDir(),
		StoryName:    testStoryName,
		ProjectHost:  "github.com",
		ProjectPath:  "kalbasit/swm",
		WorktreePath: "/tmp/stories/feat-x/github.com/kalbasit/swm",
		RepoPath:     "/tmp/repositories/github.com/kalbasit/swm",
		ConfigHome:   configHome,
	}

	require.NoError(t, hookexec.Run(context.Background(), cfg))

	logBytes, err := os.ReadFile(logFile) //nolint:gosec // G304: test-controlled path
	require.NoError(t, err)

	log := string(logBytes)
	require.Contains(t, log, "SWM_HOOK=pre-worktree-create")
	require.Contains(t, log, "SWM_STORY=feat-x")
	require.Contains(t, log, "SWM_PROJECT_HOST=github.com")
	require.Contains(t, log, "SWM_PROJECT_PATH=kalbasit/swm")
}

func TestRun_StdinJSON(t *testing.T) {
	t.Parallel()

	configHome := t.TempDir()

	logFile := filepath.Join(t.TempDir(), "stdin.log")
	// Script reads stdin into the log file.
	script := "cat > " + logFile + "\n"

	globalDir := filepath.Join(configHome, "swm", "hooks")
	installScript(t, globalDir, "post-clone", "00-check", script)

	cfg := hookexec.RunConfig{
		Event:       "post-clone",
		CodeRoot:    t.TempDir(),
		StoryName:   testStoryName,
		ProjectHost: "github.com",
		ProjectPath: "kalbasit/swm",
		ConfigHome:  configHome,
	}

	require.NoError(t, hookexec.Run(context.Background(), cfg))

	logBytes, err := os.ReadFile(logFile) //nolint:gosec // G304: test-controlled path
	require.NoError(t, err)

	log := string(logBytes)
	require.Contains(t, log, `"hook":"post-clone"`)
	require.Contains(t, log, `"story":"feat-x"`)
	require.Contains(t, log, `"project_host":"github.com"`)
}
