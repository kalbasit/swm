package config

import (
	"fmt"

	"github.com/spf13/cobra"

	appconfig "github.com/kalbasit/swm/cmd/swm/internal/config"
)

// NewGetCmd builds the `swm config get <key>` command.
func NewGetCmd(cfg *appconfig.Config) *cobra.Command {
	return &cobra.Command{
		Use:          "get <key>",
		Short:        "Print the effective value of a config key",
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			k, ok := appconfig.LookupKey(args[0])
			if !ok {
				return fmt.Errorf("%q: %w", args[0], appconfig.ErrUnknownKey)
			}

			// Explicitly stdout. cobra's Println writes to OutOrStderr(), which
			// sent the value to stderr and made `root=$(swm config get
			// code_root)` come back empty -- the only thing this command is
			// for. A steward host agent hit exactly that and reported the code
			// root as missing on a machine where it was configured.
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), k.Get(cfg)); err != nil {
				return fmt.Errorf("writing value: %w", err)
			}

			return nil
		},
	}
}
