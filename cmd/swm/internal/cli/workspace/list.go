package workspace

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"
)

// NewListCmd returns the `swm workspace list` command.
func NewListCmd(store coreStory.Store, defaultStory string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all workspaces and their attached projects",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			stories, err := store.List(cmd.Context())
			if err != nil {
				return fmt.Errorf("listing workspaces: %w", err)
			}

			sort.Slice(stories, func(i, j int) bool {
				return stories[i].Name < stories[j].Name
			})

			renderWorkspaceTree(cmd.OutOrStdout(), stories, defaultStory)

			return nil
		},
	}
}

// renderWorkspaceTree writes a two-level tree of workspaces and their projects to w,
// skipping the default story. Projects are rendered with box-drawing glyphs (├──/└──).
func renderWorkspaceTree(w io.Writer, stories []*coreStory.Story, defaultStory string) {
	for _, s := range stories {
		if s.Name == defaultStory {
			continue
		}

		fmt.Fprintf(w, "%s\n", s.Name) //nolint:errcheck // writing to output; errors are non-actionable

		if len(s.Projects) == 0 {
			continue
		}

		paths := make([]string, 0, len(s.Projects))
		for _, p := range s.Projects {
			paths = append(paths, p.Host+"/"+strings.Join(p.Segments, "/"))
		}

		sort.Strings(paths)

		for i, path := range paths {
			glyph := "├── "
			if i == len(paths)-1 {
				glyph = "└── "
			}

			fmt.Fprintf(w, "%s%s\n", glyph, path) //nolint:errcheck // writing to output; errors are non-actionable
		}
	}
}
