package cmd

import (
	"github.com/spf13/cobra"
)

var codeVcsCmd = &cobra.Command{
	Use:   "vcs",
	Short: "Interact with repositories available locally in the code directory",
}

func init() {
	codeCmd.AddCommand(codeVcsCmd)
}
