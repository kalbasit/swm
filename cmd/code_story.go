package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var codeStoryCmd = &cobra.Command{
	Use:   "story",
	Short: "Manage stories",
}

func init() {
	codeCmd.AddCommand(codeStoryCmd)

	if err := viper.BindPFlags(codeStoryCmd.Flags()); err != nil {
		panic(fmt.Sprintf("error binding cobra flags to viper: %s", err))
	}
}
