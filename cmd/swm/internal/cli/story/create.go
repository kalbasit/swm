// Package story contains the `swm story` sub-commands.
package story

import (
	"fmt"

	"github.com/spf13/cobra"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"

	"github.com/kalbasit/swm/cmd/swm/internal/hookexec"
)

// NewCreateCmd returns the `swm story create` command.
func NewCreateCmd(store coreStory.Store, codeRoot string, hooks hookexec.Runner) *cobra.Command {
	var branch string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new story",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ctx := cmd.Context()

			if branch == "" {
				branch = "feat/" + name
			}

			preCfg := hookexec.RunConfig{
				Event:     "pre-story-create",
				CodeRoot:  codeRoot,
				StoryName: name,
			}

			if err := hooks.Run(ctx, preCfg); err != nil {
				return fmt.Errorf("pre-story-create hook: %w", err)
			}

			if _, err := store.Create(ctx, name, branch); err != nil {
				return fmt.Errorf("creating story %q: %w", name, err)
			}

			postCfg := hookexec.RunConfig{
				Event:     "post-story-create",
				CodeRoot:  codeRoot,
				StoryName: name,
			}

			if err := hooks.Run(ctx, postCfg); err != nil {
				return fmt.Errorf("post-story-create hook: %w", err)
			}

			cmd.Printf("created story %q with branch %q\n", name, branch)

			return nil
		},
	}

	cmd.Flags().StringVar(&branch, "branch", "", "branch name (default: feat/<name>)")

	return cmd
}
