package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var codeCmd = &cobra.Command{
	Use:   "code",
	Short: "Manage the code directory",
}

func init() {
	rootCmd.AddCommand(codeCmd)

	codeCmd.Flags().String("github-access-token", "", "The access token for accessing Github")
	if err := viper.BindPFlag("github-access-token", codeCmd.Flags().Lookup("github-access-token")); err != nil {
		panic(err)
	}

	codeCmd.Flags().String("exclude", "", "The pattern to exclude")
	if err := viper.BindPFlag("exclude", codeCmd.Flags().Lookup("exclude")); err != nil {
		panic(err)
	}
}
