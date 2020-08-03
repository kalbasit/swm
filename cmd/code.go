package cmd

import (
	"fmt"

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
	codeCmd.Flags().String("exclude", "", "The pattern to exclude")

	if err := viper.BindPFlags(codeCmd.Flags()); err != nil {
		panic(fmt.Sprintf("error binding cobra flags to viper: %s", err))
	}
}
