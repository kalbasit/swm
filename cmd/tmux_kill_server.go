package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var tmuxKillServerCmd = &cobra.Command{
	Use:     "kill-server",
	Short:   "Kill the server closes the tmux session for this profile and story",
	PreRunE: tmuxPreRunE,
	RunE:    tmuxKillServerRun,
}

func init() {
	tmuxCmd.AddCommand(tmuxKillServerCmd)

	tmuxKillServerCmd.Flags().String("story-name", "", "The name of the story")
	tmuxKillServerCmd.Flags().String("story-branch-name", "", "The name of the branch. By default, it's set the same as the story-name")
	tmuxKillServerCmd.Flags().Bool("vim-exit", false, "if vim is found running, kill it")

	if err := viper.BindPFlags(tmuxKillServerCmd.Flags()); err != nil {
		panic(fmt.Sprintf("error binding cobra flags to viper: %s", err))
	}
}

func tmuxKillServerRun(cmd *cobra.Command, args []string) error {
	return tmuxManager.KillServer(viper.GetBool("vim-exit"))
}
