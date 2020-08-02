package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the configuration file",
	RunE:  initRun,
}

func init() {
	rootCmd.AddCommand(initCmd)

	if err := viper.BindPFlags(initCmd.Flags()); err != nil {
		panic(fmt.Sprintf("error binding cobra flags to viper: %s", err))
	}
}

func initRun(cmd *cobra.Command, args []string) error {
	return nil
}
