// Package integration contains end-to-end tests for swm using real plugin binaries.
package integration_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var (
	vcsGitBin      string
	sessionTmuxBin string
	faketmuxBin    string
	pickerFzfBin   string
	fakefzfBin     string
	forgeGithubBin string
)

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "swm-integration-*")
	if err != nil {
		panic("create temp dir: " + err.Error())
	}
	defer os.RemoveAll(tmpDir) //nolint:errcheck // best-effort cleanup in TestMain

	// Clear any inherited hostsvc socket so plugin processes connect to the
	// test's own hostsvc rather than a stale socket from an enclosing swm session.
	os.Unsetenv("SWM_HOST_SOCKET") //nolint:errcheck // Unsetenv cannot fail on valid key

	// Compute the repo root relative to this test file.
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "../../../../"))

	// Use pre-built binaries from env vars when available (e.g. in the Nix sandbox),
	// falling back to go build for local development.
	getBin := func(envVar, binName, pkgPath string) string {
		if p := os.Getenv(envVar); p != "" {
			return p
		}

		bin := filepath.Join(tmpDir, binName)
		if err := buildBin(repoRoot, bin, filepath.Join(repoRoot, pkgPath)); err != nil {
			panic(fmt.Sprintf("build %s: %v", binName, err))
		}

		return bin
	}

	vcsGitBin = getBin("SWM_PLUGIN_VCS_GIT_BIN", "swm-plugin-vcs-git", "plugins/vcs-git")
	sessionTmuxBin = getBin("SWM_PLUGIN_SESSION_TMUX_BIN", "swm-plugin-session-tmux", "plugins/session-tmux")
	faketmuxBin = getBin("SWM_TEST_FAKETMUX_BIN", "tmux", "plugins/session-tmux/internal/session/testdata/faketmux")
	pickerFzfBin = getBin("SWM_PLUGIN_PICKER_FZF_BIN", "swm-plugin-picker-fzf", "plugins/picker-fzf")
	fakefzfBin = getBin("SWM_TEST_FAKEFZF_BIN", "fzf", "plugins/picker-fzf/internal/picker/testdata/fakefzf")
	forgeGithubBin = getBin("SWM_PLUGIN_FORGE_GITHUB_BIN", "swm-plugin-forge-github", "plugins/forge-github")

	os.Exit(m.Run())
}

// buildBin compiles a Go package from pkgDir into outBin.
// When go.work exists at the repo root it is used for cross-module resolution;
// otherwise each module's replace directives handle it.
func buildBin(repoRoot, outBin, pkgDir string) error {
	cmd := exec.Command("go", "build", "-o", outBin, ".") //nolint:gosec // building from trusted repo paths
	cmd.Dir = pkgDir

	// Strip any existing GOWORK override from the inherited environment.
	env := make([]string, 0, len(os.Environ()))
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "GOWORK=") {
			continue
		}

		env = append(env, e)
	}

	// Only set GOWORK when the file actually exists; the plugin go.mod files
	// carry replace directives that resolve local modules without a workspace.
	goWorkPath := filepath.Join(repoRoot, "go.work")
	if _, err := os.Stat(goWorkPath); err == nil {
		env = append(env, "GOWORK="+goWorkPath)
	}

	cmd.Env = env

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w\n%s", err, out)
	}

	return nil
}
