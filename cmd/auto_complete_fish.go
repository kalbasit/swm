package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var autoCompleteFishCmd = &cobra.Command{
	Use:   "fish",
	Short: "Generate Fish auto-completion",
	RunE:  autoCompleteFishRun,
}

func init() {
	autoCompleteCmd.AddCommand(autoCompleteFishCmd)
}

func autoCompleteFishRun(cmd *cobra.Command, args []string) error {
	return cmd.Root().GenFishCompletion(os.Stdout, true)
}
