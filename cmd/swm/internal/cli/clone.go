package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
)

// errUnexpectedVCSPlugin is returned when the vcs plugin is not the expected type.
var errUnexpectedVCSPlugin = errors.New("unexpected vcs plugin type")

// NewCloneCmd returns the `swm clone` command.
func NewCloneCmd(mgr PluginManager, resolver *layout.Resolver) *cobra.Command {
	return &cobra.Command{
		Use:   "clone <url>",
		Short: "Clone a repository to its canonical path",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			ctx := cmd.Context()

			raw, err := mgr.Get(ctx, "vcs")
			if err != nil {
				return fmt.Errorf("loading vcs plugin: %w", err)
			}

			vcs, ok := raw.(pluginv1.VCSClient)
			if !ok {
				return fmt.Errorf("%w: %T", errUnexpectedVCSPlugin, raw)
			}

			id, err := vcs.ParseRemoteURL(ctx, &pluginv1.ParseRemoteURLRequest{Url: url})
			if err != nil {
				return fmt.Errorf("parsing URL %q: %w", url, err)
			}

			canonical := resolver.CanonicalPath(id)

			if _, err := os.Stat(filepath.Join(canonical, ".git")); err == nil {
				cmd.Printf("already cloned at %s\n", canonical)

				return nil
			}

			if _, err := vcs.Clone(ctx, &pluginv1.CloneRequest{
				Url:             url,
				DestinationPath: canonical,
			}); err != nil {
				return fmt.Errorf("cloning %q: %w", url, err)
			}

			cmd.Printf("cloned to %s\n", canonical)

			return nil
		},
	}
}
