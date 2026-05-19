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

			cmd.Println(k.Get(cfg))

			return nil
		},
	}
}
