// Package story contains the `swm story` sub-commands.
package story

import (
	"fmt"

	"github.com/spf13/cobra"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"
)

// NewCreateCmd returns the `swm story create` command.
func NewCreateCmd(store coreStory.Store) *cobra.Command {
	var branch string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new story",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if branch == "" {
				branch = "feat/" + name
			}

			if _, err := store.Create(cmd.Context(), name, branch); err != nil {
				return fmt.Errorf("creating story %q: %w", name, err)
			}

			cmd.Printf("created story %q with branch %q\n", name, branch)

			return nil
		},
	}

	cmd.Flags().StringVar(&branch, "branch", "", "branch name (default: feat/<name>)")

	return cmd
}
