// Package cli contains the swm CLI commands.
package cli

import (
	"context"

	"github.com/spf13/cobra"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"

	"github.com/kalbasit/swm/cmd/swm/internal/cli/story"
	"github.com/kalbasit/swm/cmd/swm/internal/cli/workspace"
	"github.com/kalbasit/swm/cmd/swm/internal/config"
	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
)

// PluginManager is the interface the CLI uses to retrieve plugin clients.
type PluginManager interface {
	Get(ctx context.Context, capability string) (any, error)
}

// NewRootCmd builds the top-level swm cobra.Command.
func NewRootCmd(
	cfg *config.Config,
	mgr PluginManager,
	store coreStory.Store,
	resolver *layout.Resolver,
) *cobra.Command {
	root := &cobra.Command{
		Use:          "swm",
		Short:        "Story-based Workflow Manager",
		SilenceUsage: true,
	}

	storyGroup := &cobra.Command{Use: "story", Short: "Manage stories"}
	storyGroup.AddCommand(story.NewCreateCmd(store))
	storyGroup.AddCommand(story.NewRemoveCmd(store, mgr, resolver))
	root.AddCommand(storyGroup)

	root.AddCommand(NewCloneCmd(mgr, resolver))

	wsGroup := &cobra.Command{Use: "workspace", Short: "Manage workspaces"}
	wsGroup.AddCommand(workspace.NewOpenCmd(cfg, store, mgr, resolver))
	root.AddCommand(wsGroup)

	return root
}
