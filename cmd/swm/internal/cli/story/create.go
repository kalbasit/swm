// Package story contains the `swm story` sub-commands.
package story

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"

	"github.com/kalbasit/swm/cmd/swm/internal/hookexec"
)

// CreateWithHooks runs pre-story-create hooks, creates the story with the given
// branch, then runs post-story-create hooks. Post-hook failure is logged but
// does not abort (the story was already created successfully).
func CreateWithHooks(
	ctx context.Context, store coreStory.Store, hooks hookexec.Runner, codeRoot, name, branch string,
) error {
	preCfg := hookexec.RunConfig{
		Event:     "pre-story-create",
		CodeRoot:  codeRoot,
		StoryName: name,
		WorkDir:   codeRoot,
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
		WorkDir:   codeRoot,
	}
	if err := hooks.Run(ctx, postCfg); err != nil {
		slog.WarnContext(ctx, "post-story-create hook failed", "err", err)
	}

	return nil
}

// NewCreateCmd returns the `swm story create` command.
func NewCreateCmd(
	store coreStory.Store,
	codeRoot string,
	hooks hookexec.Runner,
	branchNameTemplate string,
) *cobra.Command {
	var branch string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new story",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ctx := cmd.Context()

			if branch == "" {
				derived, err := branchFromTemplate(branchNameTemplate, name)
				if err != nil {
					return err
				}

				branch = derived
			}

			if err := CreateWithHooks(ctx, store, hooks, codeRoot, name, branch); err != nil {
				return err
			}

			cmd.Printf("created story %q with branch %q\n", name, branch)

			return nil
		},
	}

	cmd.Flags().StringVar(&branch, "branch", "", "branch name (default: derived from config branch_name_template)")

	return cmd
}
