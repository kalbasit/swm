package cmd

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var tmuxKillServerCmd = &cobra.Command{
	Use:     "kill-server",
	Short:   "Kill the server closes the tmux session for this profile and story",
	PreRunE: tmuxPreRunE,
	RunE:    tmuxKillServerRun,
}

func init() {
	tmuxCmd.AddCommand(tmuxKillServerCmd)

	tmuxKillServerCmd.Flags().String("story-name", os.Getenv("SWM_STORY_NAME"), "The name of the story")
	tmuxKillServerCmd.Flags().Bool("vim-exit", false, "if vim is found running, kill it")
}

func tmuxKillServerRun(cmd *cobra.Command, args []string) error {
	ve, err := cmd.Flags().GetBool("vim-exit")
	if err != nil {
		return errors.Wrap(err, "error getting the value of the --vim-exit flag")
	}

	return tmuxManager.KillServer(ve)
}
