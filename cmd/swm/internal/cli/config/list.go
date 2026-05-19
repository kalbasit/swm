package config

import (
	"fmt"

	"github.com/spf13/cobra"

	appconfig "github.com/kalbasit/swm/cmd/swm/internal/config"
)

// NewListCmd builds the `swm config list` command.
// Without --all, only keys explicitly present in the config file are shown.
// With --all, all registered keys with their effective values are shown.
func NewListCmd(cfgPath string, cfg *appconfig.Config) *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:          "list",
		Short:        "List config key-value pairs",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if all {
				return printAll(cmd, cfg)
			}

			return printConfigured(cmd, cfgPath, cfg)
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "show all registered keys with their effective values")

	return cmd
}

func printAll(cmd *cobra.Command, cfg *appconfig.Config) error {
	for _, k := range appconfig.AllKeys(cfg) {
		cmd.Printf("%s = %s\n", k.Path, k.Get(cfg))
	}

	return nil
}

func printConfigured(cmd *cobra.Command, cfgPath string, cfg *appconfig.Config) error {
	keys, err := appconfig.ConfiguredKeys(cfgPath, cfg)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	for _, k := range keys {
		cmd.Printf("%s = %s\n", k.Path, k.Get(cfg))
	}

	return nil
}
