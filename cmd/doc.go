package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var docCmd = &cobra.Command{
	Use:   "doc",
	Short: "Generate documentation for the command",
}

func init() {
	rootCmd.AddCommand(docCmd)

	if err := viper.BindPFlags(docCmd.Flags()); err != nil {
		panic(fmt.Sprintf("error binding cobra flags to viper: %s", err))
	}
}
