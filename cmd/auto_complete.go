package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var autoCompleteCmd = &cobra.Command{
	Use:   "auto-complete",
	Short: "Generate auto-completion for your shell",
}

func init() {
	rootCmd.AddCommand(autoCompleteCmd)

	if err := viper.BindPFlags(autoCompleteCmd.Flags()); err != nil {
		panic(fmt.Sprintf("error binding cobra flags to viper: %s", err))
	}
}
