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

	if p := os.Getenv("SWM_PLUGIN_VCS_GIT_BIN"); p != "" {
		vcsGitBin = p
	} else {
		vcsGitBin = filepath.Join(tmpDir, "swm-plugin-vcs-git")
		if err := buildBin(repoRoot, vcsGitBin, filepath.Join(repoRoot, "plugins/vcs-git")); err != nil {
			panic("build vcs-git: " + err.Error())
		}
	}

	if p := os.Getenv("SWM_PLUGIN_SESSION_TMUX_BIN"); p != "" {
		sessionTmuxBin = p
	} else {
		sessionTmuxBin = filepath.Join(tmpDir, "swm-plugin-session-tmux")
		if err := buildBin(repoRoot, sessionTmuxBin, filepath.Join(repoRoot, "plugins/session-tmux")); err != nil {
			panic("build session-tmux: " + err.Error())
		}
	}

	if p := os.Getenv("SWM_TEST_FAKETMUX_BIN"); p != "" {
		faketmuxBin = p
	} else {
		faketmuxBin = filepath.Join(tmpDir, "tmux")

		faketmuxSrc := filepath.Join(repoRoot, "plugins/session-tmux/internal/session/testdata/faketmux")
		if err := buildBin(repoRoot, faketmuxBin, faketmuxSrc); err != nil {
			panic("build faketmux: " + err.Error())
		}
	}

	if p := os.Getenv("SWM_PLUGIN_PICKER_FZF_BIN"); p != "" {
		pickerFzfBin = p
	} else {
		pickerFzfBin = filepath.Join(tmpDir, "swm-plugin-picker-fzf")
		if err := buildBin(repoRoot, pickerFzfBin, filepath.Join(repoRoot, "plugins/picker-fzf")); err != nil {
			panic("build picker-fzf: " + err.Error())
		}
	}

	if p := os.Getenv("SWM_TEST_FAKEFZF_BIN"); p != "" {
		fakefzfBin = p
	} else {
		fakefzfBin = filepath.Join(tmpDir, "fzf")

		fakefzfSrc := filepath.Join(repoRoot, "plugins/picker-fzf/internal/picker/testdata/fakefzf")
		if err := buildBin(repoRoot, fakefzfBin, fakefzfSrc); err != nil {
			panic("build fakefzf: " + err.Error())
		}
	}

	if p := os.Getenv("SWM_PLUGIN_FORGE_GITHUB_BIN"); p != "" {
		forgeGithubBin = p
	} else {
		forgeGithubBin = filepath.Join(tmpDir, "swm-plugin-forge-github")
		if err := buildBin(repoRoot, forgeGithubBin, filepath.Join(repoRoot, "plugins/forge-github")); err != nil {
			panic("build forge-github: " + err.Error())
		}
	}

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
