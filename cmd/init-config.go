package cmd

import (
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var initConfigCmd = &cobra.Command{
	Use:   "init-config",
	Short: "Initialize the configuration file",
	RunE:  initConfigRun,
}

func init() {
	rootCmd.AddCommand(initConfigCmd)
}

func initConfigRun(cmd *cobra.Command, args []string) error {
	if err := os.MkdirAll(path.Dir(configPath), 0755); err != nil {
		return errors.Wrap(err, "error creating the parent directory of the config file")
	}

	return viper.SafeWriteConfigAs(configPath)
}
