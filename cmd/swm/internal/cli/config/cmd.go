// Package config implements the `swm config` CLI subcommands.
package config

import (
	"github.com/spf13/cobra"

	appconfig "github.com/kalbasit/swm/cmd/swm/internal/config"
)

// NewConfigCmd builds the `swm config` command group.
func NewConfigCmd(cfgPath string, cfg *appconfig.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Read and write swm configuration",
	}

	cmd.AddCommand(NewGetCmd(cfg))
	cmd.AddCommand(NewSetCmd(cfgPath))
	cmd.AddCommand(NewListCmd(cfgPath, cfg))

	return cmd
}
