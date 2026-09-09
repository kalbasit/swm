package workspace

import (
	"github.com/spf13/cobra"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"
)

// storyNameCompletion completes a command's optional story-name argument with
// the names in the store, and offers nothing once that argument is given.
//
// Shared by `open` and `ensure` so the two cannot come to disagree about what a
// story name is; a store error offers no candidates rather than falling back to
// filenames, which would complete a path where a story name belongs.
func storyNameCompletion(
	store coreStory.Store,
) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		stories, err := store.List(cmd.Context())
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		names := make([]string, len(stories))
		for i, s := range stories {
			names[i] = s.Name
		}

		return names, cobra.ShellCompDirectiveNoFileComp
	}
}
