package cmd

import (
	"fmt"
	"os"

	"github.com/kalbasit/swm/story"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var errStoryIsRequired = errors.New("you must specify a story name with the --story-name flag")

var codeStoryCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new story",
	RunE:  codeStoryCreateRun,
}

func init() {
	codeStoryCmd.AddCommand(codeStoryCreateCmd)

	codeStoryCreateCmd.Flags().String("name", os.Getenv("SWM_STORY_NAME"), "The name of the story")

	codeStoryCreateCmd.Flags().String("branch-name", os.Getenv("SWM_STORY_NAME"), "The name of the branch. By default, it's set the same as the story-name")
}

func codeStoryCreateRun(cmd *cobra.Command, args []string) error {
	sn, err := cmd.Flags().GetString("name")
	if err != nil {
		return errors.Wrap(err, "error getting the value of the --story-name flag")
	}

	sbn, err := cmd.Flags().GetString("branch-name")
	if err != nil {
		return errors.Wrap(err, "error getting the value of the --story-name flag")
	}

	if err := story.Create(sn, sbn); err != nil {
		return errors.Wrap(err, "error creating the story")
	}

	fmt.Printf("The story %q was created successfully!\n", sn)

	return nil
}
