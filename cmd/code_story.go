package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var codeStoryCmd = &cobra.Command{
	Use:   "story",
	Short: "Manage stories",
}

func init() {
	codeCmd.AddCommand(codeStoryCmd)

	if err := viper.BindPFlags(codeStoryCmd.Flags()); err != nil {
		panic(fmt.Sprintf("error binding cobra flags to viper: %s", err))
	}
}

func codeStoryPrePrunE(cmd *cobra.Command, args []string) error {
	if err := requireCodePath(cmd, args); err != nil {
		return err
	}

	sn := viper.GetString("story-name")
	sbn := viper.GetString("story-branch-name")
	if sn == "" {
		return errStoryIsRequired
	}
	if sbn == "" {
		sbn = sn
	}

	code.SetStoryName(sn)
	code.SetStoryBranchName(sbn)

	return nil
}
