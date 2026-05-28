package workspace

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"
	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"
)

var (
	errCloseNoStoryName        = errors.New("story name required: pass <name> or set $SWM_STORY")
	errUnexpectedSessionPlugin = errors.New("unexpected session plugin type")
)

// NewCloseCmd returns the `swm workspace close` command.
func NewCloseCmd(store coreStory.Store, mgr pluginManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "close [<name>]",
		Short: "Close the active workspace for a story without removing the story",
		Args:  cobra.MaximumNArgs(1),
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			mgr.Warm(cmd.Context(), "session") //nolint:errcheck,gosec // Warm always returns nil; errors deferred to Get

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var name string
			if len(args) == 1 {
				name = args[0]
			} else {
				name = os.Getenv("SWM_STORY")
			}

			if name == "" {
				return errCloseNoStoryName
			}

			ctx := cmd.Context()

			raw, err := mgr.Get(ctx, "session")
			if err != nil {
				return fmt.Errorf("loading session plugin: %w", err)
			}

			sess, ok := raw.(pluginv1.SessionClient)
			if !ok {
				return fmt.Errorf("%w: expected pluginv1.SessionClient, got %T", errUnexpectedSessionPlugin, raw)
			}

			stream, err := sess.ListWorkspaces(ctx, &pluginv1.Empty{})
			if err != nil {
				return fmt.Errorf("listing workspaces: %w", err)
			}

			for {
				ws, err := stream.Recv()
				if errors.Is(err, io.EOF) {
					break
				}

				if err != nil {
					return fmt.Errorf("receiving workspace: %w", err)
				}

				if ws.GetStoryName() == name {
					if _, err := sess.CloseWorkspace(ctx, &pluginv1.CloseWorkspaceRequest{
						WorkspaceId: ws.GetWorkspaceId(),
					}); err != nil {
						return fmt.Errorf("closing workspace %q for story %q: %w", ws.GetWorkspaceId(), name, err)
					}

					cmd.Printf("closed workspace for story %q\n", name)

					return nil
				}
			}

			// No active workspace found — succeed (idempotent).
			return nil
		},
	}

	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		stories, err := store.List(cmd.Context())
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		names := make([]string, 0, len(stories))
		for _, s := range stories {
			if s != nil {
				names = append(names, s.Name)
			}
		}

		return names, cobra.ShellCompDirectiveNoFileComp
	}

	return cmd
}
