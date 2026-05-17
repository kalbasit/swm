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

			for _, s := range stories {
				if s.Name == defaultStory {
					continue
				}

				cmd.Println(s.Name)
			}

			return nil
		},
	}
}
