package cmd

import (
	"fmt"

	"github.com/kalbasit/swm/tmux"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var tmuxManager tmux.Manager

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

	sn := viper.GetString("story-name")
	sbn := viper.GetString("story-branch-name")
	if sbn == "" {
		sbn = sn
	}

	code.SetStoryName(sn)
	code.SetStoryBranchName(sbn)

	tmuxManager = tmux.New(code)

	return nil
}
