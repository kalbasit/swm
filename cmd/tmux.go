package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/kalbasit/swm/tmux"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var tmuxManager *tmux.Manager

var tmuxCmd = &cobra.Command{
	Use:   "tmux",
	Short: "Manage tmux sessions",
}

func init() {
	rootCmd.AddCommand(tmuxCmd)

	if err := viper.BindPFlags(tmuxCmd.Flags()); err != nil {
		panic(fmt.Sprintf("error binding cobra flags to viper: %s", err))
	}
}

func tmuxPreRunE(cmd *cobra.Command, args []string) error {
	if err := requireCodePath(cmd, args); err != nil {
		return err
	}

	sn, err := cmd.Flags().GetString("story-name")
	if err != nil {
		return err
	}

	if tmuxManager, err = tmux.New(code, sn); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			usageStoryRequired(sn)
			os.Exit(1)
		}
		return err
	}

	return nil
}

func usageStoryRequired(sn string) {
	c := color.New(color.FgRed).Add(color.Bold)
	c.Printf("A story is required, but none was created with the name %q. In order to create or attach TMUX sessions, you must create a session with same name. You can do so with the command: swm story create\n\n", sn)
	codeStoryCreateCmd.Help()
}
