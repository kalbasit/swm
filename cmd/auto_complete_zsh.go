package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var autoCompleteZshCmd = &cobra.Command{
	Use:   "zsh",
	Short: "Generate Zsh auto-completion",
	RunE:  autoCompleteZshRun,
}

func init() {
	autoCompleteCmd.AddCommand(autoCompleteZshCmd)

	if err := viper.BindPFlags(autoCompleteZshCmd.Flags()); err != nil {
		panic(fmt.Sprintf("error binding cobra flags to viper: %s", err))
	}
}

func autoCompleteZshRun(cmd *cobra.Command, args []string) error {
	return cmd.Root().GenZshCompletion(os.Stdout)
}
