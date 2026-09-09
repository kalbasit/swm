package workspace

import (
	"github.com/spf13/cobra"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"
)

// storyNameCompletion completes a command's optional story-name argument with
// the names in the store, and offers nothing once that argument is given.
//
// Shared by `open` and `ensure` so the two cannot come to disagree about what a
// story name is.
//
// A store error returns NoFileComp rather than Error. Cobra registers bash
// completion with `complete -o default`, and its script returns on the Error
// directive before reaching `compopt +o default` -- so Error means the shell
// completes filenames, which is the one thing that must not happen where a
// story name belongs. NoFileComp with no candidates offers nothing, which is
// what "no completions" should mean.
func storyNameCompletion(
	store coreStory.Store,
) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		stories, err := store.List(cmd.Context())
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		names := make([]string, len(stories))
		for i, s := range stories {
			names[i] = s.Name
		}

		return names, cobra.ShellCompDirectiveNoFileComp
	}
}
