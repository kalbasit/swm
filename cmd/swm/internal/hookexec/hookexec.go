// Package hookexec discovers and runs plain-executable lifecycle hooks.
// It implements the three-tier hook resolution defined in the swm TDD §6.6:
// global, per-repository, and per-story.
package hookexec

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
)

// RunConfig holds all the context needed to run hooks for a lifecycle event.
type RunConfig struct {
	// Event is the hook event name, e.g. "pre-story-create".
	Event string

	// CodeRoot is the swm code root directory.
	CodeRoot string

	// StoryName is the name of the story being operated on.
	StoryName string

	// ProjectHost is the project's host (e.g. "github.com"). Empty if not applicable.
	ProjectHost string

	// ProjectPath is the project's path segments joined by "/" (e.g. "kalbasit/swm"). Empty if not applicable.
	ProjectPath string

	// WorktreePath is the full path to the worktree. Empty if not applicable.
	WorktreePath string

	// RepoPath is the full path to the canonical repository clone. Empty if not applicable.
	RepoPath string

	// ConfigHome overrides the XDG config home used for global and per-story
	// hook tiers. When empty, xdg.ConfigHome is used. Inject in tests to avoid
	// relying on the xdg package's cached (init-time) value.
	ConfigHome string
}

// Runner can execute lifecycle hooks for a given event.
type Runner interface {
	Run(ctx context.Context, cfg RunConfig) error
}

// RunnerFunc is a function that implements Runner.
type RunnerFunc func(ctx context.Context, cfg RunConfig) error

// Run satisfies Runner.
func (f RunnerFunc) Run(ctx context.Context, cfg RunConfig) error { return f(ctx, cfg) }

// Noop is a Runner that always succeeds without executing any hooks.
//
//nolint:gochecknoglobals // package-level sentinel for tests and no-op wiring
var Noop Runner = RunnerFunc(func(_ context.Context, _ RunConfig) error { return nil })

// Run discovers and executes hooks for the given event across all three tiers.
// For pre-* events, a non-zero hook exit aborts immediately and returns an error.
// For post-* events, failures are logged but do not affect the return value.
func Run(ctx context.Context, cfg RunConfig) error {
	isPre := strings.HasPrefix(cfg.Event, "pre-")

	tiers := hookTiers(cfg)

	for _, tier := range tiers {
		hooks, err := findHooks(tier)
		if err != nil {
			// Unreadable tier directories are treated as empty.
			slog.WarnContext(ctx, "hookexec: cannot read hook tier", "path", tier, "err", err)

			continue
		}

		for _, hookPath := range hooks {
			if runErr := runHook(ctx, hookPath, cfg); runErr != nil {
				if isPre {
					return fmt.Errorf("pre-hook %q failed: %w", hookPath, runErr)
				}

				slog.WarnContext(ctx, "hookexec: post-hook failed (ignored)", "path", hookPath, "err", runErr)
			}
		}
	}

	return nil
}

// hookTiers returns the directory paths for all three hook tiers, in order.
// Tiers with empty paths (e.g. per-repo when projectHost is empty) are excluded.
func hookTiers(cfg RunConfig) []string {
	eventDir := cfg.Event + ".d"

	configHome := cfg.ConfigHome
	if configHome == "" {
		configHome = xdg.ConfigHome
	}

	// 1. Global: $XDG_CONFIG_HOME/swm/hooks/<event>.d/
	global := filepath.Join(configHome, "swm", "hooks", eventDir)

	tiers := []string{global}

	// 2. Per-repo: <codeRoot>/repositories/<projectHost>/<projectPath>/.swm/hooks/<event>.d/
	// Only included when project context is present.
	if cfg.ProjectHost != "" && cfg.ProjectPath != "" {
		perRepo := filepath.Join(cfg.CodeRoot, "repositories", cfg.ProjectHost, cfg.ProjectPath, ".swm", "hooks", eventDir)
		tiers = append(tiers, perRepo)
	}

	// 3. Per-story: $XDG_CONFIG_HOME/swm/stories/<storyName>/hooks/<event>.d/
	if cfg.StoryName != "" {
		perStory := filepath.Join(configHome, "swm", "stories", cfg.StoryName, "hooks", eventDir)
		tiers = append(tiers, perStory)
	}

	return tiers
}

// findHooks returns all executable files in dir, sorted lexically.
// Returns nil (no error) if the directory does not exist.
func findHooks(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	var hooks []string

	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		info, err := e.Info()
		if err != nil {
			continue
		}

		if info.Mode()&fs.ModePerm&0o111 == 0 {
			continue
		}

		hooks = append(hooks, filepath.Join(dir, e.Name()))
	}

	return hooks, nil
}

// runHook executes a single hook binary with the appropriate env and stdin.
func runHook(ctx context.Context, hookPath string, cfg RunConfig) error {
	cmd := exec.CommandContext(ctx, hookPath)

	// Inherit the current environment and add the SWM_* vars.
	cmd.Env = append(
		os.Environ(),
		"SWM_HOOK="+cfg.Event,
		"SWM_STORY="+cfg.StoryName,
		"SWM_PROJECT_HOST="+cfg.ProjectHost,
		"SWM_PROJECT_PATH="+cfg.ProjectPath,
		"SWM_WORKTREE_PATH="+cfg.WorktreePath,
		"SWM_REPO_PATH="+cfg.RepoPath,
	)

	// Write stdin JSON in a goroutine so hooks that ignore stdin don't block.
	stdinJSON, _ := json.Marshal(map[string]string{ //nolint:errcheck // map with string values never fails to marshal
		"hook":          cfg.Event,
		"story":         cfg.StoryName,
		"project_host":  cfg.ProjectHost,
		"project_path":  cfg.ProjectPath,
		"worktree_path": cfg.WorktreePath,
		"repo_path":     cfg.RepoPath,
	})

	cmd.Stdin = bytes.NewReader(stdinJSON)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
