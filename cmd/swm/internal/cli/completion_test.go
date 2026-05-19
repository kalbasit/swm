package cli_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kalbasit/swm/cmd/swm/internal/cli"
	"github.com/kalbasit/swm/cmd/swm/internal/config"
	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
)

func TestCompletionCmd(t *testing.T) {
	t.Parallel()

	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		t.Run(shell, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			root := cli.NewRootCmd(config.Defaults(), &stubMgr{}, nil, layout.NewResolver("", ""))
			root.SetOut(&buf)
			root.SetArgs([]string{"completion", shell})

			require.NoError(t, root.Execute())
			require.NotEmpty(t, buf.String())
		})
	}
}
