package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var genDocCmd = &cobra.Command{
	Use:   "gen-doc",
	Short: "Generate documentation for the command",
}

func init() {
	rootCmd.AddCommand(genDocCmd)

	if err := viper.BindPFlags(genDocCmd.Flags()); err != nil {
		panic(fmt.Sprintf("error binding cobra flags to viper: %s", err))
	}
}
