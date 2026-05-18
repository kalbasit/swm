package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kalbasit/swm/cmd/swm/internal/config"
)

func TestLoad_FileNotFound(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load("/nonexistent/path/config.toml")
	require.ErrorIs(t, err, config.ErrConfigNotFound)
	require.Nil(t, cfg)
}

func TestLoad_Defaults(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	require.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte(""), 0o600))

	cfg, err := config.Load(path)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(home, "code"), cfg.CodeRoot)
	require.Equal(t, "_default", cfg.DefaultStory)
}

func TestLoad_TildeInCodeRoot(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	require.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte(`code_root = "~/mycode"`), 0o600))

	cfg, err := config.Load(path)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(home, "mycode"), cfg.CodeRoot)
}

func TestLoad_TildeInPluginPaths(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	require.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
code_root = "/code"

[plugins.paths]
vcs-git = "~/bin/swm-plugin-vcs-git"
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cfg, err := config.Load(path)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(home, "bin", "swm-plugin-vcs-git"), cfg.Plugins.Paths["vcs-git"])
}

func TestLoad_AllFields(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
code_root = "/mycode"
default_story = "main"

[plugins]
session = "tmux"
vcs = "git"
picker = "fzf"
forges = ["github"]

[plugins.config.vcs-git]
foo = "bar"
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cfg, err := config.Load(path)
	require.NoError(t, err)
	require.Equal(t, "/mycode", cfg.CodeRoot)
	require.Equal(t, "main", cfg.DefaultStory)
	require.Equal(t, "tmux", cfg.Plugins.Session)
	require.Equal(t, "git", cfg.Plugins.VCS)
	require.Equal(t, "fzf", cfg.Plugins.Picker)
	require.Equal(t, []string{"github"}, cfg.Plugins.Forges)
	require.Contains(t, cfg.Plugins.Config, "vcs-git")
}

func TestLoad_MissingOptionalFields(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte(`code_root = "/code"`), 0o600))

	cfg, err := config.Load(path)
	require.NoError(t, err)
	require.Equal(t, "/code", cfg.CodeRoot)
	require.Equal(t, "_default", cfg.DefaultStory)
	require.Empty(t, cfg.Plugins.Session)
}

func TestResolveConfigPath_EnvVarSet(t *testing.T) {
	t.Parallel()

	got := config.ResolveConfigPath("/explicit/config.toml", "/home/user/.config")
	require.Equal(t, "/explicit/config.toml", got)
}

func TestResolveConfigPath_EnvVarEmpty(t *testing.T) {
	t.Parallel()

	got := config.ResolveConfigPath("", "/home/user/.config")
	require.Equal(t, filepath.Join("/home/user/.config", "swm", "config.toml"), got)
}

func TestLoad_StoryBranchNameTemplate_Default(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte(`code_root = "/code"`), 0o600))

	cfg, err := config.Load(path)
	require.NoError(t, err)
	require.Equal(t, "feat/{{.Name}}", cfg.Story.BranchNameTemplate)
}

func TestLoad_StoryBranchNameTemplate_Explicit(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
code_root = "/code"

[story]
branch_name_template = "fix/{{.Name}}"
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cfg, err := config.Load(path)
	require.NoError(t, err)
	require.Equal(t, "fix/{{.Name}}", cfg.Story.BranchNameTemplate)
}

func TestLoad_StoryBranchNameTemplate_EmptyString(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `
code_root = "/code"

[story]
branch_name_template = ""
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cfg, err := config.Load(path)
	require.NoError(t, err)
	require.Empty(t, cfg.Story.BranchNameTemplate)
}

func TestLoad_BadTOML(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte("this is not = valid [ toml"), 0o600))

	_, err := config.Load(path)
	require.Error(t, err)
	require.NotErrorIs(t, err, config.ErrConfigNotFound)
}
