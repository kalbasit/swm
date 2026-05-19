package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kalbasit/swm/cmd/swm/internal/config"
)

func TestSave_CreatesParentDirs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "config.toml")

	cfg := &config.Config{}
	cfg.CodeRoot = testCodeRoot

	require.NoError(t, config.Save(path, cfg))
	require.FileExists(t, path)
}

func TestSave_WritesExpectedContent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cfg := &config.Config{}
	cfg.CodeRoot = testCodeRoot
	cfg.Plugins.Session = testValTmux

	require.NoError(t, config.Save(path, cfg))

	data, err := os.ReadFile(path) //nolint:gosec // user-controlled test temp path, safe in tests
	require.NoError(t, err)
	require.Contains(t, string(data), "code_root")
	require.Contains(t, string(data), testCodeRoot)
	require.Contains(t, string(data), "session")
	require.Contains(t, string(data), testValTmux)
}

func TestSave_OmitsZeroValueFields(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cfg := &config.Config{}
	cfg.CodeRoot = testCodeRoot

	require.NoError(t, config.Save(path, cfg))

	data, err := os.ReadFile(path) //nolint:gosec // user-controlled test temp path, safe in tests
	require.NoError(t, err)

	// Fields not set must not appear in the output
	require.NotContains(t, string(data), "default_story")
	require.NotContains(t, string(data), "session")
	require.NotContains(t, string(data), "vcs")
}

func TestLoadForWrite_ReturnsEmptyForMissingFile(t *testing.T) {
	t.Parallel()

	cfg, err := config.LoadForWrite("/nonexistent/path/config.toml")
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Empty(t, cfg.CodeRoot)
}

func TestLoadForWrite_ReadsExistingFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte(`code_root = "/workspace"`+"\n"), 0o600))

	cfg, err := config.LoadForWrite(path)
	require.NoError(t, err)
	require.Equal(t, testCodeRoot, cfg.CodeRoot)
}

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	// Write via LoadForWrite + Save
	cfg, err := config.LoadForWrite(path)
	require.NoError(t, err)

	cfg.Plugins.Session = testValTmux
	cfg.DefaultStory = "main"
	require.NoError(t, config.Save(path, cfg))

	// Read back via Load (which applies defaults and expands tildes)
	loaded, err := config.Load(path)
	require.NoError(t, err)
	require.Equal(t, testValTmux, loaded.Plugins.Session)
	require.Equal(t, "main", loaded.DefaultStory)
	// code_root was not set, so Load's default applies (expanded to absolute path)
	require.NotEmpty(t, loaded.CodeRoot)
}
