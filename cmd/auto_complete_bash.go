package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var autoCompleteBashCmd = &cobra.Command{
	Use:   "bash",
	Short: "Generate Bash auto-completion",
	RunE:  autoCompleteBashRun,
}

func init() {
	autoCompleteCmd.AddCommand(autoCompleteBashCmd)
}

func autoCompleteBashRun(cmd *cobra.Command, args []string) error {
	return cmd.Root().GenBashCompletion(os.Stdout)
}
