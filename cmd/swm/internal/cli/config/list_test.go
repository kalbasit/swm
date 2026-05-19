package config_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	cliconfig "github.com/kalbasit/swm/cmd/swm/internal/cli/config"

	"github.com/kalbasit/swm/cmd/swm/internal/config"
)

func TestListCmd_NoFile(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	cmd := cliconfig.NewListCmd("/nonexistent/path/config.toml", cfg)

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())
	require.Empty(t, out.String())
}

func TestListCmd_ConfiguredKeysOnly(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := "code_root = \"/workspace\"\n\n[plugins]\nsession = \"tmux\"\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cfg := config.Defaults()
	cfg.CodeRoot = "/workspace"
	cfg.Plugins.Session = "tmux"

	cmd := cliconfig.NewListCmd(path, cfg)
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	require.Contains(t, lines, "code_root = /workspace")
	require.Contains(t, lines, "plugins.session = tmux")

	// Keys not in the file must not appear
	for _, line := range lines {
		require.NotContains(t, line, "default_story")
		require.NotContains(t, line, "plugins.vcs")
	}
}

func TestListCmd_ArrayKeyInline(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := "[plugins]\nforges = [\"github\"]\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cfg := config.Defaults()
	cfg.Plugins.Forges = []string{"github"}

	cmd := cliconfig.NewListCmd(path, cfg)
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())

	require.Contains(t, out.String(), `plugins.forges = ["github"]`)
}

func TestListCmd_AllFlag_NoFile(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	cmd := cliconfig.NewListCmd("/nonexistent/path/config.toml", cfg)

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetArgs([]string{testFlagAll})
	require.NoError(t, cmd.Execute())

	output := out.String()
	require.Contains(t, output, "code_root")
	require.Contains(t, output, "default_story")
	require.Contains(t, output, "plugins.session")
	require.Contains(t, output, "plugins.forges")
	require.Contains(t, output, "story.branch_name_template")
}

func TestListCmd_AllFlag_StableOrder(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	cmd1 := cliconfig.NewListCmd("/nonexistent/path/config.toml", cfg)
	cmd2 := cliconfig.NewListCmd("/nonexistent/path/config.toml", cfg)

	out1 := new(bytes.Buffer)
	cmd1.SetOut(out1)
	cmd1.SetArgs([]string{"--all"})
	require.NoError(t, cmd1.Execute())

	out2 := new(bytes.Buffer)
	cmd2.SetOut(out2)
	cmd2.SetArgs([]string{"--all"})
	require.NoError(t, cmd2.Execute())

	require.Equal(t, out1.String(), out2.String())
}

func TestListCmd_AllFlag_ShowsConfiguredAndDefault(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte("code_root = \"/workspace\"\n"), 0o600))

	cfg := config.Defaults()
	cfg.CodeRoot = "/workspace"

	cmd := cliconfig.NewListCmd(path, cfg)
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetArgs([]string{testFlagAll})
	require.NoError(t, cmd.Execute())

	output := out.String()
	require.Contains(t, output, "code_root = /workspace")
	// default_story still shows (from defaults)
	require.Contains(t, output, "default_story = _default")
}
