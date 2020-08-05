package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/kalbasit/swm/story"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var codeStoryRemoveCmd = &cobra.Command{
	Use:     "remove",
	Short:   "Remove a new story",
	PreRunE: func(cmd *cobra.Command, args []string) error { return createCode() },
	RunE:    codeStoryRemoveRun,
}

func init() {
	codeStoryCmd.AddCommand(codeStoryRemoveCmd)

	codeStoryRemoveCmd.Flags().String("name", os.Getenv("SWM_STORY_NAME"), "The name of the story")
	codeStoryRemoveCmd.Flags().Bool("force", false, "Force remove the story")
}

func codeStoryRemoveRun(cmd *cobra.Command, args []string) error {
	sn, err := cmd.Flags().GetString("name")
	if err != nil {
		return errors.Wrap(err, "error getting the value of the --story-name flag")
	}
	if sn == "" {
		return errStoryIsRequired
	}

	// ask the user for confirmation unless the force flag was given
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		return errors.Wrap(err, "error getting the value of the --force flag")
	}

	if !force {
		tty := bufio.NewReader(os.Stdin)
		fmt.Printf("Are you sure you want to remove the story %q and all its files? ", sn)
		text, err := tty.ReadString('\n')
		if err != nil {
			return errors.Wrap(err, "error reading your input")
		}

		ans := strings.TrimSpace(text)
		if !strings.EqualFold(ans, "y") && !strings.EqualFold(ans, "yes") {
			fmt.Println("Ok not removing the story")
			return nil
		}
	}

	s, err := story.Load(sn)
	if err != nil {
		return errors.Wrap(err, "error loading the story")
	}

	if err := os.RemoveAll(path.Join(code.StoriesDir(), s.GetName())); err != nil {
		return errors.Wrap(err, "error removing the files for the story")
	}

	if err := s.Remove(); err != nil {
		return errors.Wrap(err, "error removing the story")
	}

	fmt.Printf("The story %q was removed successfully!\n", sn)

	return nil
}
