package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var autoCompleteBashCmd = &cobra.Command{
	Use:   "bash",
	Short: "Generate Bash auto-completion",
	RunE:  autoCompleteBashRun,
}

func init() {
	autoCompleteCmd.AddCommand(autoCompleteBashCmd)

	if err := viper.BindPFlags(autoCompleteBashCmd.Flags()); err != nil {
		panic(fmt.Sprintf("error binding cobra flags to viper: %s", err))
	}
}

func autoCompleteBashRun(cmd *cobra.Command, args []string) error {
	return cmd.Root().GenBashCompletion(os.Stdout)
}
