package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var autoCompletePowerShellCmd = &cobra.Command{
	Use:   "power-shell",
	Short: "Generate PowerShell auto-completion",
	RunE:  autoCompletePowerShellRun,
}

func init() {
	autoCompleteCmd.AddCommand(autoCompletePowerShellCmd)

	if err := viper.BindPFlags(autoCompletePowerShellCmd.Flags()); err != nil {
		panic(fmt.Sprintf("error binding cobra flags to viper: %s", err))
	}
}

func autoCompletePowerShellRun(cmd *cobra.Command, args []string) error {
	return cmd.Root().GenPowerShellCompletion(os.Stdout)
}
