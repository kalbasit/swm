// Package pr contains the `swm pr` sub-commands.
package pr

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"
	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"
)

var errNoStoryName = errors.New("story name required: pass --story or set SWM_STORY")

// forgeManager is the subset of the plugin manager used by pr commands.
type forgeManager interface {
	GetForge(ctx context.Context, hostname string) (pluginv1.ForgeClient, error)
}

// NewListCmd returns the `swm pr list` command.
func NewListCmd(store coreStory.Store, mgr forgeManager) *cobra.Command {
	var storyName string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List open pull requests for the current story",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			if storyName == "" {
				storyName = os.Getenv("SWM_STORY")
			}

			if storyName == "" {
				return errNoStoryName
			}

			s, err := store.Get(ctx, storyName)
			if err != nil {
				return fmt.Errorf("loading story %q: %w", storyName, err)
			}

			return listPRs(ctx, cmd.OutOrStdout(), mgr, s)
		},
	}

	cmd.Flags().StringVarP(&storyName, "story", "s", "", "story name (default: $SWM_STORY)")

	return cmd
}

func listPRs(ctx context.Context, out io.Writer, mgr forgeManager, s *coreStory.Story) error {
	for _, proj := range s.Projects {
		forge, err := mgr.GetForge(ctx, proj.Host)
		if err != nil {
			// No forge configured for this host — skip silently per spec.
			continue
		}

		projectID := &pluginv1.ProjectID{
			Host:     proj.Host,
			Segments: proj.Segments,
		}

		stream, err := forge.ListPullRequests(ctx, &pluginv1.ListPRsRequest{
			ProjectId: projectID,
		})
		if err != nil {
			return fmt.Errorf("listing pull requests for %s/%s: %w",
				proj.Host, strings.Join(proj.Segments, "/"), err)
		}

		for {
			pr, err := stream.Recv()
			if err != nil {
				if isStreamDone(err) {
					break
				}

				return fmt.Errorf("receiving pull request: %w", err)
			}

			//nolint:errcheck // output write errors are non-actionable
			fmt.Fprintf(out, "#%d\t%s\t%s\n", pr.GetNumber(), pr.GetTitle(), pr.GetUrl())
		}
	}

	return nil
}

// isStreamDone reports whether err signals a normally-closed server-side stream.
func isStreamDone(err error) bool {
	return err == io.EOF
}
