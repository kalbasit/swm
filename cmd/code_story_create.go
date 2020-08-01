package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var codeStoryCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new story",
}

func init() {
	codeStoryCmd.AddCommand(codeStoryCreateCmd)

	if err := viper.BindPFlags(codeStoryCreateCmd.Flags()); err != nil {
		panic(fmt.Sprintf("error binding cobra flags to viper: %s", err))
	}
}
