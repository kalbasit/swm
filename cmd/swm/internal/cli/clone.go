package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
	"github.com/kalbasit/swm/cmd/swm/internal/hookexec"
)

// errUnexpectedVCSPlugin is returned when the vcs plugin is not the expected type.
var errUnexpectedVCSPlugin = errors.New("unexpected vcs plugin type")

// NewCloneCmd returns the `swm clone` command.
func NewCloneCmd(mgr PluginManager, resolver *layout.Resolver, hooks hookexec.Runner) *cobra.Command {
	return &cobra.Command{
		Use:   "clone <url>",
		Short: "Clone a repository to its canonical path",
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			mgr.Warm(cmd.Context(), "vcs") //nolint:errcheck,gosec // Warm always returns nil; errors deferred to Get

			return nil
		},
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

			projectPath := strings.Join(id.GetSegments(), "/")
			codeRoot := resolver.CodeRoot()

			if err := hooks.Run(ctx, hookexec.RunConfig{
				Event:       "pre-clone",
				CodeRoot:    codeRoot,
				ProjectHost: id.GetHost(),
				ProjectPath: projectPath,
				WorkDir:     codeRoot,
			}); err != nil {
				return fmt.Errorf("pre-clone hook: %w", err)
			}

			stream, err := vcs.Clone(ctx, &pluginv1.CloneRequest{
				Url:             url,
				DestinationPath: canonical,
			})
			if err != nil {
				return fmt.Errorf("cloning %q: %w", url, err)
			}

			for {
				evt, recvErr := stream.Recv()
				if errors.Is(recvErr, io.EOF) {
					break
				}

				if recvErr != nil {
					return fmt.Errorf("cloning %q: %w", url, recvErr)
				}

				if line := evt.GetProgressLine(); line != "" {
					fmt.Fprintln(cmd.ErrOrStderr(), line) //nolint:errcheck // writing progress to stderr is best-effort
				}
			}

			if err := hooks.Run(ctx, hookexec.RunConfig{
				Event:       "post-clone",
				CodeRoot:    codeRoot,
				ProjectHost: id.GetHost(),
				ProjectPath: projectPath,
				RepoPath:    canonical,
				WorkDir:     canonical,
			}); err != nil {
				// post-clone hooks are informational; log the failure but don't abort.
				cmd.PrintErrf("post-clone hook failed (ignored): %v\n", err)
			}

			cmd.Printf("cloned to %s\n", canonical)

			return nil
		},
	}
}
