package pr

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
)

var errNotInCodeRoot = errors.New("current directory is not under the swm code root")

// NewCreateCmd returns the `swm pr create` command.
func NewCreateCmd(mgr forgeManager, resolver *layout.Resolver) *cobra.Command {
	var (
		title      string
		body       string
		base       string
		headBranch string
		draft      bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a pull request for the current project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting working directory: %w", err)
			}

			pid := resolver.ProjectIDFromPath(cwd)
			if pid == nil {
				return fmt.Errorf("%w: %q", errNotInCodeRoot, cwd)
			}

			forge, err := mgr.GetForge(ctx, pid.GetHost())
			if err != nil {
				return fmt.Errorf("no forge plugin for %q: %w", pid.GetHost(), err)
			}

			pr, err := forge.CreatePullRequest(ctx, &pluginv1.CreatePRRequest{
				ProjectId:  pid,
				Title:      title,
				Body:       body,
				BaseBranch: base,
				HeadBranch: headBranch,
				Draft:      draft,
			})
			if err != nil {
				return fmt.Errorf("creating pull request: %w", err)
			}

			//nolint:errcheck // output write errors are non-actionable
			fmt.Fprintln(cmd.OutOrStdout(), pr.GetUrl())

			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "pull request title (required)")
	cmd.Flags().StringVar(&body, "body", "", "pull request body")
	cmd.Flags().StringVar(&base, "base", "main", "base branch")
	cmd.Flags().StringVar(&headBranch, "head", "", "head branch")
	cmd.Flags().BoolVar(&draft, "draft", false, "create as draft pull request")

	if err := cmd.MarkFlagRequired("title"); err != nil {
		// MarkFlagRequired only fails when the flag does not exist, which cannot
		// happen here since the flag is defined above.
		//nolint:forbidigo // invariant: flag defined above; unreachable unless flag name is mistyped
		panic(fmt.Sprintf("marking title flag required: %v", err))
	}

	return cmd
}
