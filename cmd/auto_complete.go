package cmd

import (
	"github.com/spf13/cobra"
)

var autoCompleteCmd = &cobra.Command{
	Use:   "auto-complete",
	Short: "Generate auto-completion for your shell",
}

func init() {
	rootCmd.AddCommand(autoCompleteCmd)
}
