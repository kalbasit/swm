package config

import (
	"fmt"

	"github.com/spf13/cobra"

	appconfig "github.com/kalbasit/swm/cmd/swm/internal/config"
)

// NewSetCmd builds the `swm config set <key> <value>` command.
func NewSetCmd(cfgPath string) *cobra.Command {
	return &cobra.Command{
		Use:          "set <key> <value>",
		Short:        "Write a config value to the config file",
		SilenceUsage: true,
		Args:         cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			key, value := args[0], args[1]

			k, ok := appconfig.LookupKey(key)
			if !ok {
				return fmt.Errorf("%q: %w", key, appconfig.ErrUnknownKey)
			}

			cfg, err := appconfig.LoadForWrite(cfgPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			if err := k.Set(cfg, value); err != nil {
				return err
			}

			if err := appconfig.Save(cfgPath, cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			return nil
		},
	}
}
