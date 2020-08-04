package cmd

import (
	"github.com/spf13/cobra"
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
}

func tmuxVimExitRun(cmd *cobra.Command, args []string) error {
	return tmuxManager.VimExit()
}
