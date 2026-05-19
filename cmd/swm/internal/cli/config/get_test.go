package config_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	cliconfig "github.com/kalbasit/swm/cmd/swm/internal/cli/config"

	"github.com/kalbasit/swm/cmd/swm/internal/config"
)

func TestGetCmd_KnownKeyWithDefault(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	cmd := cliconfig.NewGetCmd(cfg)

	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetArgs([]string{"default_story"})

	require.NoError(t, cmd.Execute())
	require.Equal(t, "_default\n", out.String())
}

func TestGetCmd_KnownKeyConfigured(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	cfg.Plugins.Session = testValTmux

	cmd := cliconfig.NewGetCmd(cfg)
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetArgs([]string{testKeyPluginsSession})

	require.NoError(t, cmd.Execute())
	require.Equal(t, testValTmux+"\n", out.String())
}

func TestGetCmd_NestedMapKey(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	cfg.Plugins.Paths = map[string]string{
		"vcs-git": testPluginBinPath,
	}

	cmd := cliconfig.NewGetCmd(cfg)
	out := new(bytes.Buffer)
	cmd.SetOut(out)
	cmd.SetArgs([]string{testKeyPluginPaths})

	require.NoError(t, cmd.Execute())
	require.Equal(t, testPluginBinPath+"\n", out.String())
}

func TestGetCmd_UnknownKeyExitsNonZero(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	cmd := cliconfig.NewGetCmd(cfg)
	cmd.SetArgs([]string{"does.not.exist"})

	require.Error(t, cmd.Execute())
}

func TestGetCmd_MissingArgExitsNonZero(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	cmd := cliconfig.NewGetCmd(cfg)
	cmd.SetArgs([]string{})

	require.Error(t, cmd.Execute())
}
