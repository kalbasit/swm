package story

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"
	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
)

// showOutput is the --json shape.
type showOutput struct {
	Name       string        `json:"name"`
	BranchName string        `json:"branch_name"`
	CreatedAt  time.Time     `json:"created_at"`
	Projects   []showProject `json:"projects"`
}

// showProject is one attached project and where it lives for this story.
type showProject struct {
	Key          string `json:"key"`
	WorktreePath string `json:"worktree_path"`
}

// NewShowCmd returns the `swm story show` command.
//
// It reports what swm records about one story: its branch name and where each
// attached project resolves to on this host. Both facts existed only inside the
// process before this, which left a caller automating swm to reconstruct them
// -- reading a branch with `git rev-parse`, which answers what is checked out
// now rather than what the story is on, or composing paths from the code root,
// which copies swm's layout into the caller.
//
// It reads records, not the world: no session plugin, no filesystem check. That
// is what makes it answer for a story whose workspace is closed, and on a
// machine where the multiplexer is not running.
func NewShowCmd(store coreStory.Store, resolver *layout.Resolver, defaultStory string) *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "show [<story-name>]",
		Short: "Report a story's branch name and where its projects live",
		Long: "Report one story: its branch name, when it was created, and every " +
			"attached project with the worktree path it resolves to on this host.\n\n" +
			"Paths come from swm's own records, not from the filesystem: a reported " +
			"path is where the worktree belongs, which is not a promise that it is " +
			"there. Nothing about a running session is reported -- a story exists " +
			"whether or not its workspace is open.\n\n" +
			"If [story-name] is omitted, the command falls back to the $SWM_STORY " +
			"environment variable, and then to the default story configured in swm.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			storyName := ""
			if len(args) > 0 {
				storyName = args[0]
			}

			if storyName == "" {
				storyName = os.Getenv("SWM_STORY")
			}

			if storyName == "" {
				storyName = defaultStory
			}

			st, err := store.Get(cmd.Context(), storyName)
			if err != nil {
				return fmt.Errorf("loading story %q: %w", storyName, err)
			}

			out := showOutput{
				Name:       st.Name,
				BranchName: st.BranchName,
				CreatedAt:  st.CreatedAt,
				Projects:   make([]showProject, 0, len(st.Projects)),
			}

			for i := range st.Projects {
				p := &st.Projects[i]
				pid := &pluginv1.ProjectID{Host: p.Host, Segments: p.Segments}

				out.Projects = append(out.Projects, showProject{
					Key: p.Host + "/" + strings.Join(p.Segments, "/"),
					// Asked rather than composed. The resolver already knows
					// that the default story has no separate worktree and
					// resolves to the canonical clone; rebuilding that
					// condition here would be a second place for it to be
					// wrong.
					WorktreePath: resolver.WorktreePath(storyName, pid),
				})
			}

			return printStory(cmd, out, asJSON)
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "print the story as a JSON object")

	cmd.ValidArgsFunction = storyNameCompletion(store)

	return cmd
}

func printStory(cmd *cobra.Command, out showOutput, asJSON bool) error {
	if asJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		if err := enc.Encode(out); err != nil {
			return fmt.Errorf("encoding json: %w", err)
		}

		return nil
	}

	// Explicitly stdout: cobra's Print/Println go to stderr, which would put
	// this out of reach of a caller capturing output.
	w := cmd.OutOrStdout()

	var b strings.Builder

	fmt.Fprintf(&b, "%s\n", out.Name)
	fmt.Fprintf(&b, "  branch  %s\n", out.BranchName)
	// Absolute rather than an age. This is the form a caller pastes into a
	// bug report or compares against a log line, and "3d ago" stops being
	// true the moment it is written down.
	fmt.Fprintf(&b, "  created %s\n", out.CreatedAt.Format(time.RFC3339))

	for _, p := range out.Projects {
		fmt.Fprintf(&b, "  %s -> %s\n", p.Key, p.WorktreePath)
	}

	if _, err := io.WriteString(w, b.String()); err != nil {
		return fmt.Errorf("writing story: %w", err)
	}

	return nil
}
