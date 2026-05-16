// Package workspace contains the `swm workspace` sub-commands.
package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"
	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/cmd/swm/internal/config"
	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
)

// pluginManager is the subset of the CLI plugin manager used by this command.
type pluginManager interface {
	Get(ctx context.Context, capability string) (any, error)
}

// errUnexpectedPluginType is returned when the plugin is not the expected client type.
var errUnexpectedPluginType = errors.New("unexpected plugin type")

// NewOpenCmd returns the `swm workspace open` command.
func NewOpenCmd(
	cfg *config.Config,
	store coreStory.Store,
	mgr pluginManager,
	resolver *layout.Resolver,
) *cobra.Command {
	var storyName string

	cmd := &cobra.Command{
		Use:   "open",
		Short: "Open (or attach to) the workspace for a story",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			if storyName == "" {
				storyName = os.Getenv("SWM_STORY")
			}

			if storyName == "" {
				storyName = cfg.DefaultStory
			}

			st, err := store.Get(ctx, storyName)
			if err != nil {
				if errors.Is(err, coreStory.ErrStoryNotFound) {
					return fmt.Errorf("%w: %s", coreStory.ErrStoryNotFound, storyName)
				}

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

			worktreePaths := make(map[string]string, len(st.Projects))
			for i := range st.Projects {
				p := &st.Projects[i]
				pid := &pluginv1.ProjectID{Host: p.Host, Segments: p.Segments}

				key := p.Host + "/" + strings.Join(p.Segments, "/")

				worktreePaths[key] = resolver.WorktreePath(storyName, pid)
			}

			if _, err := sess.OpenWorkspace(ctx, &pluginv1.OpenWorkspaceRequest{
				StoryName:     storyName,
				WorktreePaths: worktreePaths,
			}); err != nil {
				return fmt.Errorf("opening workspace: %w", err)
			}

			cmd.Printf("workspace opened for story %q\n", storyName)

			return nil
		},
	}

	cmd.Flags().StringVar(&storyName, "story", "", "story name (default: $SWM_STORY or default story)")

	return cmd
}
