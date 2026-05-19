package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	cliconfig "github.com/kalbasit/swm/cmd/swm/internal/cli/config"

	"github.com/kalbasit/swm/cmd/swm/internal/config"
)

func TestSetCmd_CreatesNewFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cmd := cliconfig.NewSetCmd(path)
	cmd.SetArgs([]string{testKeyPluginsSession, testValTmux})
	require.NoError(t, cmd.Execute())

	loaded, err := config.Load(path)
	require.NoError(t, err)
	require.Equal(t, testValTmux, loaded.Plugins.Session)
}

func TestSetCmd_UpdatesExistingFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte("code_root = \"/workspace\"\n"), 0o600))

	cmd := cliconfig.NewSetCmd(path)
	cmd.SetArgs([]string{testKeyPluginsSession, testValTmux})
	require.NoError(t, cmd.Execute())

	loaded, err := config.Load(path)
	require.NoError(t, err)
	require.Equal(t, "/workspace", loaded.CodeRoot)
	require.Equal(t, testValTmux, loaded.Plugins.Session)
}

func TestSetCmd_MapSubkey(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cmd := cliconfig.NewSetCmd(path)
	cmd.SetArgs([]string{testKeyPluginPaths, testPluginBinPath})
	require.NoError(t, cmd.Execute())

	loaded, err := config.Load(path)
	require.NoError(t, err)
	require.Equal(t, testPluginBinPath, loaded.Plugins.Paths["vcs-git"])
}

func TestSetCmd_NonWritableKeyExitsNonZero(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cmd := cliconfig.NewSetCmd(path)
	cmd.SetArgs([]string{"plugins.forges", "github"})
	require.Error(t, cmd.Execute())
}

func TestSetCmd_UnknownKeyExitsNonZero(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cmd := cliconfig.NewSetCmd(path)
	cmd.SetArgs([]string{"does.not.exist", "value"})
	require.Error(t, cmd.Execute())
}

func TestSetCmd_MissingValueArgExitsNonZero(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cmd := cliconfig.NewSetCmd(path)
	cmd.SetArgs([]string{"plugins.session"})
	require.Error(t, cmd.Execute())
}

func TestSetCmd_CreatesParentDirs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "config.toml")

	cmd := cliconfig.NewSetCmd(path)
	cmd.SetArgs([]string{"default_story", "main"})
	require.NoError(t, cmd.Execute())
	require.FileExists(t, path)
}
