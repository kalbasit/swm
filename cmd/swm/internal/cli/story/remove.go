// Package story contains the `swm story` sub-commands.
package story

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"
	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
	"github.com/kalbasit/swm/cmd/swm/internal/hookexec"
)

// pluginManager is the subset of the CLI plugin manager used by this command.
type pluginManager interface {
	Get(ctx context.Context, capability string) (any, error)
	Warm(ctx context.Context, capabilities ...string) error
}

// errRemovalFailed is returned when one or more steps during story removal fail.
var errRemovalFailed = errors.New("removal failed")

// errNoStoryName is returned when no story name is provided and $SWM_STORY is unset.
var errNoStoryName = errors.New("story name required: pass <name> or set $SWM_STORY")

// NewRemoveCmd returns the `swm story remove` command.
func NewRemoveCmd(
	store coreStory.Store,
	mgr pluginManager,
	resolver *layout.Resolver,
	hooks hookexec.Runner,
) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "remove [<name>]",
		Short: "Remove a story and all its worktrees",
		Args:  cobra.MaximumNArgs(1),
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			// Warm session concurrently but discard its error — session is optional;
			// RunE silently falls back when it is absent (closeStoryWorkspace is best-effort).
			var wg sync.WaitGroup

			wg.Go(func() {
				_ = mgr.Warm(cmd.Context(), "session") //nolint:errcheck // session is optional
			})

			err := mgr.Warm(cmd.Context(), "vcs")

			wg.Wait()

			return err
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var name string
			if len(args) == 1 {
				name = args[0]
			} else {
				name = os.Getenv("SWM_STORY")
			}

			if name == "" {
				return errNoStoryName
			}

			ctx := cmd.Context()

			st, err := store.Get(ctx, name)
			if err != nil {
				if errors.Is(err, coreStory.ErrStoryNotFound) {
					return fmt.Errorf("%w: %s", coreStory.ErrStoryNotFound, name)
				}

				return fmt.Errorf("loading story %q: %w", name, err)
			}

			if !force && len(st.Projects) > 0 {
				cmd.Printf("Story %q has %d attached project(s):\n", name, len(st.Projects))

				for _, p := range st.Projects {
					cmd.Printf("  - %s/%s\n", p.Host, joinSegments(p.Segments))
				}

				cmd.Printf("Continue? [y/N]: ")

				var resp string

				if _, err := fmt.Fscan(cmd.InOrStdin(), &resp); err != nil {
					cmd.Println("aborted")

					return nil //nolint:nilerr // scan failure (e.g. EOF) is an intentional abort, not a caller error
				}

				resp = strings.ToLower(strings.TrimSpace(resp))
				if resp != "y" && resp != "yes" {
					cmd.Println("aborted")

					return nil
				}
			}

			return removeStory(ctx, cmd, name, st, mgr, store, resolver, hooks)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation prompt")

	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
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

	return cmd
}

func removeStory(
	ctx context.Context,
	cmd *cobra.Command,
	name string,
	st *coreStory.Story,
	mgr pluginManager,
	store coreStory.Store,
	resolver *layout.Resolver,
	hooks hookexec.Runner,
) error {
	codeRoot := resolver.CodeRoot()

	if err := hooks.Run(ctx, hookexec.RunConfig{
		Event:     "pre-story-remove",
		CodeRoot:  codeRoot,
		StoryName: name,
		WorkDir:   codeRoot,
	}); err != nil {
		return fmt.Errorf("pre-story-remove hook: %w", err)
	}

	var errs []error

	// Remove all worktrees — best-effort, collect failures.
	if len(st.Projects) > 0 {
		raw, err := mgr.Get(ctx, "vcs")
		if err != nil {
			errs = append(errs, fmt.Errorf("loading vcs plugin: %w", err))
		} else if vcs, ok := raw.(pluginv1.VCSClient); ok {
			for i := range st.Projects {
				p := &st.Projects[i]
				pid := &pluginv1.ProjectID{Host: p.Host, Segments: p.Segments}
				worktreePath := resolver.WorktreePath(name, pid)
				repoPath := resolver.CanonicalPath(pid)
				projectPath := strings.Join(p.Segments, "/")

				preWT := hookexec.RunConfig{
					Event:        "pre-worktree-remove",
					CodeRoot:     codeRoot,
					StoryName:    name,
					ProjectHost:  p.Host,
					ProjectPath:  projectPath,
					WorktreePath: worktreePath,
					RepoPath:     repoPath,
					WorkDir:      worktreePath,
				}

				if err := hooks.Run(ctx, preWT); err != nil {
					slog.WarnContext(ctx, "pre-worktree-remove hook failed (continuing)", "err", err)
				}

				if _, err := vcs.RemoveWorktree(ctx, &pluginv1.RemoveWorktreeRequest{
					WorktreePath: worktreePath,
				}); err != nil {
					if status.Code(err) != codes.NotFound {
						errs = append(errs, fmt.Errorf("removing worktree %s: %w", worktreePath, err))
					}
				}

				postWT := hookexec.RunConfig{
					Event:        "post-worktree-remove",
					CodeRoot:     codeRoot,
					StoryName:    name,
					ProjectHost:  p.Host,
					ProjectPath:  projectPath,
					WorktreePath: worktreePath,
					RepoPath:     repoPath,
					WorkDir:      repoPath,
				}

				if err := hooks.Run(ctx, postWT); err != nil {
					slog.WarnContext(ctx, "post-worktree-remove hook failed (continuing)", "err", err)
				}
			}
		}
	}

	// Close workspace — best-effort.
	if raw, err := mgr.Get(ctx, "session"); err == nil {
		if sess, ok := raw.(pluginv1.SessionClient); ok {
			closeStoryWorkspace(ctx, sess, name)
		}
	}

	// Delete story JSON.
	if err := store.Delete(ctx, name); err != nil {
		errs = append(errs, fmt.Errorf("deleting story: %w", err))
	}

	if len(errs) > 0 {
		for _, e := range errs {
			cmd.PrintErrf("error: %v\n", e)
		}

		return fmt.Errorf("%w: %d error(s)", errRemovalFailed, len(errs))
	}

	if err := hooks.Run(ctx, hookexec.RunConfig{
		Event:     "post-story-remove",
		CodeRoot:  codeRoot,
		StoryName: name,
		WorkDir:   codeRoot,
	}); err != nil {
		slog.WarnContext(ctx, "post-story-remove hook failed", "err", err)
	}

	cmd.Printf("removed story %q\n", name)

	return nil
}

// closeStoryWorkspace finds and closes the workspace for the given story (best-effort).
func closeStoryWorkspace(ctx context.Context, sess pluginv1.SessionClient, storyName string) {
	stream, err := sess.ListWorkspaces(ctx, &pluginv1.Empty{})
	if err != nil {
		return
	}

	for {
		ws, err := stream.Recv()
		if err != nil {
			return
		}

		if ws.GetStoryName() == storyName {
			_, _ = sess.CloseWorkspace(ctx, &pluginv1.CloseWorkspaceRequest{ //nolint:errcheck // best-effort close
				WorkspaceId: ws.GetWorkspaceId(),
			})

			return
		}
	}
}

func joinSegments(segs []string) string {
	return strings.Join(segs, "/")
}
