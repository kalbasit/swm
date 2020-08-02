package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var codePullRequestCmd = &cobra.Command{
	Use:     "pull-request",
	Aliases: []string{"pr"},
	Short:   "Pull request sub-command provides commands to interact with Github",
}

func init() {
	codeCmd.AddCommand(codePullRequestCmd)

	if err := viper.BindPFlags(codePullRequestCmd.Flags()); err != nil {
		panic(fmt.Sprintf("error binding cobra flags to viper: %s", err))
	}
}
