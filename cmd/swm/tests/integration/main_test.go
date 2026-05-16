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
)

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "swm-integration-*")
	if err != nil {
		panic("create temp dir: " + err.Error())
	}
	defer os.RemoveAll(tmpDir) //nolint:errcheck // best-effort cleanup in TestMain

	// Compute the repo root relative to this test file.
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "../../../../"))

	// Compile vcs-git plugin.
	vcsGitBin = filepath.Join(tmpDir, "swm-plugin-vcs-git")
	if err := buildBin(repoRoot, vcsGitBin, filepath.Join(repoRoot, "plugins/vcs-git")); err != nil {
		panic("build vcs-git: " + err.Error())
	}

	// Compile session-tmux plugin.
	sessionTmuxBin = filepath.Join(tmpDir, "swm-plugin-session-tmux")
	if err := buildBin(repoRoot, sessionTmuxBin, filepath.Join(repoRoot, "plugins/session-tmux")); err != nil {
		panic("build session-tmux: " + err.Error())
	}

	// Compile faketmux binary (from session-tmux testdata).
	faketmuxBin = filepath.Join(tmpDir, "tmux")

	faketmuxSrc := filepath.Join(repoRoot, "plugins/session-tmux/internal/session/testdata/faketmux")
	if err := buildBin(repoRoot, faketmuxBin, faketmuxSrc); err != nil {
		panic("build faketmux: " + err.Error())
	}

	os.Exit(m.Run())
}

// buildBin compiles a Go package from pkgDir into outBin.
// It strips GOWORK=off from the environment so the workspace is discoverable.
func buildBin(repoRoot, outBin, pkgDir string) error {
	cmd := exec.Command("go", "build", "-o", outBin, ".") //nolint:gosec // building from trusted repo paths
	cmd.Dir = pkgDir

	// Inherit the full environment but remove any GOWORK=off override so
	// the workspace file at repoRoot is used for dependency resolution.
	env := make([]string, 0, len(os.Environ()))
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "GOWORK=") {
			continue
		}

		env = append(env, e)
	}

	env = append(env, "GOWORK="+filepath.Join(repoRoot, "go.work"))
	cmd.Env = env

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w\n%s", err, out)
	}

	return nil
}
