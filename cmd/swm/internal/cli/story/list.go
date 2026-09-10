package story

import (
	"fmt"

	"github.com/spf13/cobra"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"
)

// NewListCmd returns the `swm story list` command.
func NewListCmd(store coreStory.Store, defaultStory string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all stories",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			stories, err := store.List(cmd.Context())
			if err != nil {
				return fmt.Errorf("listing stories: %w", err)
			}

			// Explicitly stdout. cobra's Println writes to OutOrStderr(), which
			// left this command's stdout empty -- so `swm story list | grep`
			// found nothing and every story looked absent. A steward host agent
			// asked whether a story existed, was told no, and failed a real
			// assignment while that story's worktree sat on disk.
			out := cmd.OutOrStdout()

			for _, s := range stories {
				if s.Name == defaultStory {
					continue
				}

				if _, err := fmt.Fprintln(out, s.Name); err != nil {
					return fmt.Errorf("writing story names: %w", err)
				}
			}

			return nil
		},
	}
}
