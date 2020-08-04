package cmd

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var tmuxSwitchClientCmd = &cobra.Command{
	Use:     "switch-client",
	Short:   "Switch the client within the session for this profile and story",
	PreRunE: tmuxPreRunE,
	RunE:    tmuxSwitchClientRun,
}

func init() {
	tmuxCmd.AddCommand(tmuxSwitchClientCmd)

	tmuxSwitchClientCmd.Flags().String("story-name", os.Getenv("SWM_STORY_NAME"), "The name of the story")
	tmuxSwitchClientCmd.Flags().Bool("kill-pane", false, "kill the TMUX pane after switch client")
}

func tmuxSwitchClientRun(cmd *cobra.Command, args []string) error {
	kp, err := cmd.Flags().GetBool("kill-pane")
	if err != nil {
		return errors.Wrap(err, "error getting the value of the --kill-pane flag")
	}

	return tmuxManager.SwitchClient(kp)
}
