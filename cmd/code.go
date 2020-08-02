package cmd

import (
	"github.com/spf13/cobra"
)

var codeCmd = &cobra.Command{
	Use:   "code",
	Short: "Manage the code directory",
}

func init() {
	rootCmd.AddCommand(codeCmd)

	codeCmd.Flags().String("github-access-token", "", "The access token for accessing Github")
	codeCmd.Flags().String("exclude", "", "The pattern to exclude")
}
