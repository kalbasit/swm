package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var autoCompleteZshCmd = &cobra.Command{
	Use:   "zsh",
	Short: "Generate Zsh auto-completion",
	RunE:  autoCompleteZshRun,
}

func init() {
	autoCompleteCmd.AddCommand(autoCompleteZshCmd)
}

func autoCompleteZshRun(cmd *cobra.Command, args []string) error {
	return cmd.Root().GenZshCompletion(os.Stdout)
}
