package cmd

import (
	"github.com/spf13/cobra"
)

var codeStoryCmd = &cobra.Command{
	Use:   "story",
	Short: "Manage stories",
}

func init() {
	codeCmd.AddCommand(codeStoryCmd)
}
