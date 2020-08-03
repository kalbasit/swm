package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var tmuxSwitchClientCmd = &cobra.Command{
	Use:     "switch-client",
	Short:   "Switch the client within the session for this profile and story",
	PreRunE: tmuxPreRunE,
	RunE:    tmuxSwitchClientRun,
}

func init() {
	tmuxCmd.AddCommand(tmuxSwitchClientCmd)

	tmuxSwitchClientCmd.Flags().String("story-name", "", "The name of the story")
	tmuxSwitchClientCmd.Flags().String("story-branch-name", "", "The name of the branch. By default, it's set the same as the story-name")
	tmuxSwitchClientCmd.Flags().Bool("kill-pane", false, "kill the TMUX pane after switch client")

	if err := viper.BindPFlags(tmuxSwitchClientCmd.Flags()); err != nil {
		panic(fmt.Sprintf("error binding cobra flags to viper: %s", err))
	}
}

func tmuxSwitchClientRun(cmd *cobra.Command, args []string) error {
	return tmuxManager.SwitchClient(viper.GetBool("kill-pane"))
}
