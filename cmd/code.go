package cmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var codeCmd = &cobra.Command{
	Use:   "code",
	Short: "Manage the code directory",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := createGithubClient(); err != nil {
			return errors.Wrap(err, "error creating a GitHub client")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(codeCmd)

	codeCmd.Flags().String("github-access-token", "", "The access token for accessing Github")
	codeCmd.Flags().String("exclude", "", "The pattern to exclude")
}
