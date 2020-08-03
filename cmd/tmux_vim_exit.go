package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var tmuxVimExitCmd = &cobra.Command{
	Use:     "vim-exit",
	Short:   "Close all of open Vim within the session for this profile and story",
	PreRunE: tmuxPreRunE,
	RunE:    tmuxVimExitRun,
}

func init() {
	tmuxCmd.AddCommand(tmuxVimExitCmd)

	tmuxVimExitCmd.Flags().String("story-name", "", "The name of the story")
	tmuxVimExitCmd.Flags().String("story-branch-name", "", "The name of the branch. By default, it's set the same as the story-name")

	if err := viper.BindPFlags(tmuxVimExitCmd.Flags()); err != nil {
		panic(fmt.Sprintf("error binding cobra flags to viper: %s", err))
	}
}

func tmuxVimExitRun(cmd *cobra.Command, args []string) error {
	return tmuxManager.VimExit()
}
