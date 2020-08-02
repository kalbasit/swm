package cmd

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var errStoryIsRequired = errors.New("you must specify a story name with the --story-name flag")

var codeStoryCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   "Create a new story",
	PreRunE: codeStoryPrePrunE,
	RunE:    codeStoryCreateRun,
}

func init() {
	codeStoryCmd.AddCommand(codeStoryCreateCmd)

	codeStoryCreateCmd.Flags().String("story-name", "", "The name of the story")
	codeStoryCreateCmd.Flags().String("story-branch-name", "", "The name of the branch. By default, it's set the same as the story-name")

	if err := viper.BindPFlags(codeStoryCreateCmd.Flags()); err != nil {
		panic(fmt.Sprintf("error binding cobra flags to viper: %s", err))
	}
}

func codeStoryCreateRun(cmd *cobra.Command, args []string) error {
	return code.CreateStory()
}
