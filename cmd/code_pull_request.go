package cmd

import (
	"github.com/spf13/cobra"
)

var codePullRequestCmd = &cobra.Command{
	Use:     "pull-request",
	Aliases: []string{"pr"},
	Short:   "Pull request sub-command provides commands to interact with Github",
}

func init() {
	codeCmd.AddCommand(codePullRequestCmd)
}
