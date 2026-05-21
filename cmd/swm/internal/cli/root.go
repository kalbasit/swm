// Package cli contains the swm CLI commands.
package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	cliconfig "github.com/kalbasit/swm/cmd/swm/internal/cli/config"
	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"
	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/cmd/swm/internal/cli/pr"
	"github.com/kalbasit/swm/cmd/swm/internal/cli/story"
	"github.com/kalbasit/swm/cmd/swm/internal/cli/workspace"
	"github.com/kalbasit/swm/cmd/swm/internal/config"
	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
	"github.com/kalbasit/swm/cmd/swm/internal/hookexec"
)

// PluginManager is the interface the CLI uses to retrieve plugin clients.
type PluginManager interface {
	Get(ctx context.Context, capability string) (any, error)
	GetForge(ctx context.Context, hostname string) (pluginv1.ForgeClient, error)
	Warm(ctx context.Context, capabilities ...string) error
	Close() error
}

// NewRootCmd builds the top-level swm cobra.Command.
func NewRootCmd(
	cfgPath string,
	cfg *config.Config,
	mgr PluginManager,
	store coreStory.Store,
	resolver *layout.Resolver,
	openOpts ...workspace.OpenOption,
) *cobra.Command {
	var logLevel string

	root := &cobra.Command{
		Use:               "swm",
		Short:             "Story-based Workflow Manager",
		SilenceUsage:      true,
		CompletionOptions: cobra.CompletionOptions{HiddenDefaultCmd: true},
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			var level slog.Level
			if err := level.UnmarshalText([]byte(logLevel)); err != nil {
				return fmt.Errorf("invalid --log-level %q: %w", logLevel, err)
			}

			slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

			return nil
		},
	}

	root.PersistentFlags().StringVar(&logLevel, "log-level", "warn", "log level (debug, info, warn, error)")

	hooks := hookexec.RunnerFunc(func(ctx context.Context, rc hookexec.RunConfig) error {
		if rc.ConfigHome == "" {
			rc.ConfigHome = cfg.HooksConfigHome
		}

		return hookexec.Run(ctx, rc)
	})

	storyGroup := &cobra.Command{Use: "story", Short: "Manage stories"}
	storyGroup.AddCommand(story.NewCreateCmd(store, cfg.CodeRoot, hooks, cfg.Story.BranchNameTemplate))
	storyGroup.AddCommand(story.NewListCmd(store, cfg.DefaultStory))
	storyGroup.AddCommand(story.NewRemoveCmd(store, mgr, resolver, hooks))
	root.AddCommand(storyGroup)

	root.AddCommand(NewCloneCmd(mgr, resolver, hooks))

	wsGroup := &cobra.Command{Use: "workspace", Short: "Manage workspaces"}
	wsGroup.AddCommand(workspace.NewOpenCmd(cfg, store, mgr, resolver, hooks, openOpts...))
	wsGroup.AddCommand(workspace.NewListCmd(store, cfg.DefaultStory))
	root.AddCommand(wsGroup)

	prGroup := &cobra.Command{Use: "pr", Short: "Manage pull requests"}
	prGroup.AddCommand(pr.NewListCmd(store, mgr, cfg))
	prGroup.AddCommand(pr.NewCreateCmd(mgr, resolver, store, cfg))
	root.AddCommand(prGroup)

	root.AddCommand(cliconfig.NewConfigCmd(cfgPath, cfg))

	return root
}
