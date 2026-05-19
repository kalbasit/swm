package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kalbasit/swm/cmd/swm/internal/config"
)

func TestLookupKey_StaticKeys(t *testing.T) {
	t.Parallel()

	paths := []string{
		"code_root",
		"default_story",
		"plugins.session",
		"plugins.vcs",
		"plugins.picker",
		"plugins.forges",
		"story.branch_name_template",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			k, ok := config.LookupKey(path)
			require.True(t, ok)
			require.Equal(t, path, k.Path)
		})
	}
}

func TestLookupKey_UnknownKey(t *testing.T) {
	t.Parallel()

	_, ok := config.LookupKey("does.not.exist")
	require.False(t, ok)
}

func TestLookupKey_DynamicPathKey(t *testing.T) {
	t.Parallel()

	k, ok := config.LookupKey(testKeyPluginPaths)
	require.True(t, ok)
	require.Equal(t, testKeyPluginPaths, k.Path)
	require.True(t, k.Writable)

	cfg := config.Defaults()
	require.NoError(t, k.Set(cfg, testPluginBinPath))
	require.Equal(t, testPluginBinPath, k.Get(cfg))
}

func TestLookupKey_DynamicPathKey_EmptySubkey(t *testing.T) {
	t.Parallel()

	_, ok := config.LookupKey("plugins.paths.")
	require.False(t, ok)
}

func TestKeyDef_ForgesIsNotWritable(t *testing.T) {
	t.Parallel()

	k, ok := config.LookupKey("plugins.forges")
	require.True(t, ok)
	require.False(t, k.Writable)

	err := k.Set(config.Defaults(), "foo")
	require.Error(t, err)
}

func TestKeyDef_ScalarRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path  string
		value string
	}{
		{"code_root", testCodeRoot},
		{"default_story", "main"},
		{"plugins.session", testValTmux},
		{"plugins.vcs", testValGit},
		{"plugins.picker", testValFzf},
		{"story.branch_name_template", "fix/{{.Name}}"},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()

			k, ok := config.LookupKey(tc.path)
			require.True(t, ok)

			cfg := config.Defaults()
			require.NoError(t, k.Set(cfg, tc.value))
			require.Equal(t, tc.value, k.Get(cfg))
		})
	}
}

func TestAllKeys_StableOrder(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	first := config.AllKeys(cfg)
	second := config.AllKeys(cfg)
	require.Len(t, second, len(first))

	for i := range first {
		require.Equal(t, first[i].Path, second[i].Path)
	}
}

func TestAllKeys_IncludesDynamicPaths(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	cfg.Plugins.Paths = map[string]string{
		"vcs-git": testPluginBinPath,
	}

	keys := config.AllKeys(cfg)

	paths := make([]string, len(keys))
	for i, k := range keys {
		paths[i] = k.Path
	}

	require.Contains(t, paths, testKeyPluginPaths)
}

func TestConfiguredKeys_EmptyFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte{}, 0o600))

	cfg := config.Defaults()
	keys, err := config.ConfiguredKeys(path, cfg)
	require.NoError(t, err)
	require.Empty(t, keys)
}

func TestConfiguredKeys_NoFile(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	keys, err := config.ConfiguredKeys("/nonexistent/path/config.toml", cfg)
	require.NoError(t, err)
	require.Empty(t, keys)
}

func TestConfiguredKeys_WithFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := dir + "/config.toml"
	content := "code_root = \"/workspace\"\n\n[plugins]\nsession = \"tmux\"\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cfg := config.Defaults()
	cfg.CodeRoot = "/workspace"
	cfg.Plugins.Session = "tmux"

	keys, err := config.ConfiguredKeys(path, cfg)
	require.NoError(t, err)

	paths := make([]string, len(keys))
	for i, k := range keys {
		paths[i] = k.Path
	}

	require.Contains(t, paths, "code_root")
	require.Contains(t, paths, "plugins.session")
	require.NotContains(t, paths, "default_story")
}
