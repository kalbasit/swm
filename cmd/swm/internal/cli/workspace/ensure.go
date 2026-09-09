package workspace

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"
	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/cmd/swm/internal/config"
	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
	"github.com/kalbasit/swm/cmd/swm/internal/hookexec"
)

// ensureOutput is the --json shape: what exists after the command ran.
type ensureOutput struct {
	WorkspaceID string             `json:"workspace_id"`
	PaneGroups  []ensureOutputPane `json:"pane_groups"`
}

type ensureOutputPane struct {
	Project     string `json:"project"`
	PaneGroupID string `json:"pane_group_id"`
}

// NewEnsureCmd returns the `swm workspace ensure` command.
//
// It is `open`'s script-shaped sibling: it makes the workspace and its pane
// groups exist, prints the workspace id, and exits. It never attaches, never
// picks and never prompts, with or without a terminal -- a command whose
// behaviour depends on how it was invoked is one a caller with no terminal
// cannot rely on.
func NewEnsureCmd(
	cfg *config.Config,
	store coreStory.Store,
	mgr pluginManager,
	resolver *layout.Resolver,
	hooks hookexec.Runner,
) *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "ensure [story-name]",
		Short: "Make a story's workspace and pane groups exist, without attaching",
		Long: "Make a story's workspace and pane groups exist, then exit, printing the " +
			"workspace id on stdout and nothing else.\n\n" +
			"Unlike `swm workspace open` this never attaches to the workspace, never shows " +
			"a picker and never prompts, so it can be driven from a script or a daemon. " +
			"A story that does not exist is an error rather than a prompt: use " +
			"`swm story create` to make one.\n\n" +
			"If [story-name] is omitted, the command falls back to the $SWM_STORY " +
			"environment variable, and then to the default story configured in swm.",
		Args: cobra.MaximumNArgs(1),
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			//nolint:errcheck,gosec // Warm always returns nil; errors deferred to Get
			mgr.Warm(cmd.Context(), "session")

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			storyName := ""
			if len(args) > 0 {
				storyName = args[0]
			}

			if storyName == "" {
				storyName = os.Getenv("SWM_STORY")
			}

			if storyName == "" {
				storyName = cfg.DefaultStory
			}

			// Deliberately no prompt and no creation. A typo must not leave a
			// new story behind, and a caller that wants one has story create.
			st, err := store.Get(ctx, storyName)
			if err != nil {
				return fmt.Errorf("loading story %q: %w", storyName, err)
			}

			raw, err := mgr.Get(ctx, "session")
			if err != nil {
				return fmt.Errorf("loading session plugin: %w", err)
			}

			sess, ok := raw.(pluginv1.SessionClient)
			if !ok {
				return fmt.Errorf("%w: %T", errUnexpectedPluginType, raw)
			}

			if err := hooks.Run(ctx, hookexec.RunConfig{
				Event:     eventPreWorkspaceOpen,
				CodeRoot:  cfg.CodeRoot,
				StoryName: storyName,
				WorkDir:   cfg.CodeRoot,
			}); err != nil {
				return fmt.Errorf("pre-workspace-open hook: %w", err)
			}

			slog.DebugContext(
				ctx, "workspace ensure",
				"story", storyName,
				"projects", len(st.Projects),
			)

			// Every attached project, not just the first: this command has no
			// cursor to place and cannot know which project the next command
			// will name.
			opened, err := openWorkspaceWithGroups(ctx, sess, resolver, st, storyName, st.Projects)
			if err != nil {
				return err
			}

			// Same asymmetry as `open`: a failing post hook is logged by the
			// runner and does not change the exit status.
			workDir := cfg.CodeRoot
			if len(opened.Groups) > 0 {
				workDir = opened.WorktreePaths[opened.Groups[0].ProjectKey]
			}

			//nolint:errcheck // post-* hooks never fail the command; Run already logs
			_ = hooks.Run(ctx, hookexec.RunConfig{
				Event:     eventPostWorkspaceOpen,
				CodeRoot:  cfg.CodeRoot,
				StoryName: storyName,
				WorkDir:   workDir,
			})

			return printEnsured(cmd, opened, asJSON)
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "print the workspace and its pane groups as a JSON object")

	cmd.ValidArgsFunction = storyNameCompletion(store)

	return cmd
}

// printEnsured writes the result: by default exactly the workspace id and a
// newline, so `ws=$(swm workspace ensure x)` captures something usable.
func printEnsured(cmd *cobra.Command, opened *openedWorkspace, asJSON bool) error {
	if !asJSON {
		// Explicitly stdout: cobra's Print/Println go to stderr, and the whole
		// point of this command is that `ws=$(swm workspace ensure x)` captures
		// the id.
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), opened.ID); err != nil {
			return fmt.Errorf("writing workspace id: %w", err)
		}

		return nil
	}

	out := ensureOutput{
		WorkspaceID: opened.ID,
		PaneGroups:  make([]ensureOutputPane, 0, len(opened.Groups)),
	}

	for _, g := range opened.Groups {
		out.PaneGroups = append(out.PaneGroups, ensureOutputPane{
			Project:     g.ProjectKey,
			PaneGroupID: g.PaneGroupID,
		})
	}

	enc := json.NewEncoder(cmd.OutOrStdout())
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("encoding json: %w", err)
	}

	return nil
}
