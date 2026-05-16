// Package story contains the `swm story` sub-commands.
package story

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"
	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
)

// pluginManager is the subset of the CLI plugin manager used by this command.
type pluginManager interface {
	Get(ctx context.Context, capability string) (any, error)
}

// errRemovalFailed is returned when one or more steps during story removal fail.
var errRemovalFailed = errors.New("removal failed")

// NewRemoveCmd returns the `swm story remove` command.
func NewRemoveCmd(store coreStory.Store, mgr pluginManager, resolver *layout.Resolver) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a story and all its worktrees",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
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

				_, _ = fmt.Fscan(cmd.InOrStdin(), &resp) //nolint:errcheck // scan failure (e.g. EOF) treats as abort
				if resp != "y" {
					cmd.Println("aborted")

					return nil
				}
			}

			return removeStory(ctx, cmd, name, st, mgr, store, resolver)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation prompt")

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
) error {
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

				if _, err := vcs.RemoveWorktree(ctx, &pluginv1.RemoveWorktreeRequest{
					WorktreePath: worktreePath,
				}); err != nil {
					if status.Code(err) != codes.NotFound {
						errs = append(errs, fmt.Errorf("removing worktree %s: %w", worktreePath, err))
					}
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
