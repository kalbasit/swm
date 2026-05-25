package layout_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kalbasit/swm/plugins/session-tmux/internal/layout"
)

const minimalConfig = `
[[windows]]
name = "main"
`

const twoWindowConfig = `
[[windows]]
name = "editor"

[[windows]]
name = "shell"
`

const templateConfig = `
[[windows]]
name = "main"
path = "{{.WorktreePath}}/src"

  [[windows.panes]]
  commands = ["echo {{.StoryName}} {{.TmuxSocket}}"]
`

func writeConfig(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o700))
	require.NoError(t, os.WriteFile(p, []byte(content), 0o600))
	return p
}

func TestLoadConfig_NoConfigAtEitherTier(t *testing.T) {
	t.Parallel()

	cfg, err := layout.LoadConfig("/nonexistent/worktree", "/nonexistent/xdg", layout.TemplateVars{})
	require.NoError(t, err)
	require.Nil(t, cfg)
}

func TestLoadConfig_PerRepoConfigUsed(t *testing.T) {
	t.Parallel()

	wt := t.TempDir()
	xdg := t.TempDir()
	writeConfig(t, filepath.Join(wt, ".swm"), "session-tmux.toml", minimalConfig)

	cfg, err := layout.LoadConfig(wt, xdg, layout.TemplateVars{})
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Windows, 1)
	require.Equal(t, "main", cfg.Windows[0].Name)
}

func TestLoadConfig_GlobalConfigUsedWhenNoPerRepo(t *testing.T) {
	t.Parallel()

	wt := t.TempDir()
	xdg := t.TempDir()
	writeConfig(t, filepath.Join(xdg, "swm"), "session-tmux.toml", twoWindowConfig)

	cfg, err := layout.LoadConfig(wt, xdg, layout.TemplateVars{})
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Windows, 2)
}

func TestLoadConfig_PerRepoWinsOverGlobal(t *testing.T) {
	t.Parallel()

	wt := t.TempDir()
	xdg := t.TempDir()
	writeConfig(t, filepath.Join(wt, ".swm"), "session-tmux.toml", minimalConfig)   // 1 window
	writeConfig(t, filepath.Join(xdg, "swm"), "session-tmux.toml", twoWindowConfig) // 2 windows

	cfg, err := layout.LoadConfig(wt, xdg, layout.TemplateVars{})
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Windows, 1, "per-repo config must win over global")
}

func TestLoadConfig_TemplateSubstitution(t *testing.T) {
	t.Parallel()

	wt := t.TempDir()
	xdg := t.TempDir()
	writeConfig(t, filepath.Join(wt, ".swm"), "session-tmux.toml", templateConfig)

	vars := layout.TemplateVars{
		WorktreePath: "/home/user/code/stories/feat/github.com/org/repo",
		StoryName:    "feat-x",
		TmuxSocket:   "/run/user/1000/swm/tmux/feat-x.sock",
	}

	cfg, err := layout.LoadConfig(wt, xdg, vars)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Equal(t, "/home/user/code/stories/feat/github.com/org/repo/src", cfg.Windows[0].Path)
	require.Equal(t, "echo feat-x /run/user/1000/swm/tmux/feat-x.sock", cfg.Windows[0].Panes[0].Commands[0])
}

func TestLoadConfig_ValidationNoWindows(t *testing.T) {
	t.Parallel()

	wt := t.TempDir()
	xdg := t.TempDir()
	writeConfig(t, filepath.Join(wt, ".swm"), "session-tmux.toml", `# empty`)

	_, err := layout.LoadConfig(wt, xdg, layout.TemplateVars{})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestLoadConfig_ValidationEmptyWindowName(t *testing.T) {
	t.Parallel()

	wt := t.TempDir()
	xdg := t.TempDir()
	writeConfig(t, filepath.Join(wt, ".swm"), "session-tmux.toml", `
[[windows]]
name = ""
`)

	_, err := layout.LoadConfig(wt, xdg, layout.TemplateVars{})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestLoadConfig_ValidationFlexZero(t *testing.T) {
	t.Parallel()

	flex := 0
	wt := t.TempDir()
	xdg := t.TempDir()
	_ = flex
	writeConfig(t, filepath.Join(wt, ".swm"), "session-tmux.toml", `
[[windows]]
name = "main"

  [[windows.panes]]
  flex = 0
`)

	_, err := layout.LoadConfig(wt, xdg, layout.TemplateVars{})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestLoadConfig_ValidationTwoFocusPanes(t *testing.T) {
	t.Parallel()

	wt := t.TempDir()
	xdg := t.TempDir()
	writeConfig(t, filepath.Join(wt, ".swm"), "session-tmux.toml", `
[[windows]]
name = "main"

  [[windows.panes]]
  focus = true

  [[windows.panes]]
  focus = true
`)

	_, err := layout.LoadConfig(wt, xdg, layout.TemplateVars{})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Contains(t, err.Error(), "focus")
}

func TestLoadConfig_ValidationTwoZoomPanes(t *testing.T) {
	t.Parallel()

	wt := t.TempDir()
	xdg := t.TempDir()
	writeConfig(t, filepath.Join(wt, ".swm"), "session-tmux.toml", `
[[windows]]
name = "main"

  [[windows.panes]]
  zoom = true

  [[windows.panes]]
  zoom = true
`)

	_, err := layout.LoadConfig(wt, xdg, layout.TemplateVars{})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Contains(t, err.Error(), "zoom")
}

func TestLoadConfig_ValidationFlexNegative(t *testing.T) {
	t.Parallel()

	wt := t.TempDir()
	xdg := t.TempDir()
	writeConfig(t, filepath.Join(wt, ".swm"), "session-tmux.toml", `
[[windows]]
name = "main"

  [[windows.panes]]
  flex = -1
`)

	_, err := layout.LoadConfig(wt, xdg, layout.TemplateVars{})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}
