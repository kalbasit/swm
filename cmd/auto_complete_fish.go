package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var autoCompleteFishCmd = &cobra.Command{
	Use:   "fish",
	Short: "Generate Fish auto-completion",
	RunE:  autoCompleteFishRun,
}

func init() {
	autoCompleteCmd.AddCommand(autoCompleteFishCmd)

	if err := viper.BindPFlags(autoCompleteFishCmd.Flags()); err != nil {
		panic(fmt.Sprintf("error binding cobra flags to viper: %s", err))
	}
}

func autoCompleteFishRun(cmd *cobra.Command, args []string) error {
	return cmd.Root().GenFishCompletion(os.Stdout, true)
}
