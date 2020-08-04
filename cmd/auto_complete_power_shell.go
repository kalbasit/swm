package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var autoCompletePowerShellCmd = &cobra.Command{
	Use:   "power-shell",
	Short: "Generate PowerShell auto-completion",
	RunE:  autoCompletePowerShellRun,
}

func init() {
	autoCompleteCmd.AddCommand(autoCompletePowerShellCmd)
}

func autoCompletePowerShellRun(cmd *cobra.Command, args []string) error {
	return cmd.Root().GenPowerShellCompletion(os.Stdout)
}
