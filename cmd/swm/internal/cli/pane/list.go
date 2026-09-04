package pane

import (
	"errors"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"
)

// NewListCmd returns the `swm pane list` command.
func NewListCmd(mgr PluginManager) *cobra.Command {
	var (
		workspaceID string
		paneGroupID string
		asJSON      bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List panes",
		Long: "List panes, optionally narrowed to one workspace or one pane group.\n\n" +
			"Both filters are optional: with neither, every pane in every live " +
			"workspace is listed. That is the call to make first, because it is how a " +
			"caller learns the workspace ids the other commands require.",
		Args:    cobra.NoArgs,
		PreRunE: warmSession(mgr),
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			sess, err := sessionClient(ctx, mgr)
			if err != nil {
				return err
			}

			stream, err := sess.ListPanes(ctx, &pluginv1.ListPanesRequest{
				WorkspaceId: workspaceID,
				PaneGroupId: paneGroupID,
			})
			if err != nil {
				return fmt.Errorf("listing panes: %w", err)
			}

			// Non-nil so that an empty result encodes as an empty JSON array
			// rather than null: a caller iterating the output should not have
			// to special-case "no panes".
			panes := []paneView{}

			for {
				pane, err := stream.Recv()
				if errors.Is(err, io.EOF) {
					break
				}

				if err != nil {
					return fmt.Errorf("receiving pane: %w", err)
				}

				panes = append(panes, newPaneView(pane))
			}

			if asJSON {
				return writeJSON(cmd.OutOrStdout(), panes)
			}

			return renderPaneTable(cmd.OutOrStdout(), panes)
		},
	}

	cmd.Flags().StringVarP(&workspaceID, "workspace", "w", "",
		"only panes in this workspace (default: every live workspace)")
	cmd.Flags().StringVarP(&paneGroupID, "pane-group", "g", "",
		"only panes in this pane group (default: every pane group)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the panes as a JSON array")

	return cmd
}

// renderPaneTable writes an aligned table of panes to w.
//
// Nothing at all is written when there are no panes, not even the header, so
// that a shell loop over the output does not have to skip one.
func renderPaneTable(w io.Writer, panes []paneView) error {
	if len(panes) == 0 {
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	//nolint:errcheck // writing to output; errors surface from Flush below
	fmt.Fprintln(tw, "PANE ID\tWORKSPACE\tPANE GROUP\tFOCUSED\tCOMMAND\tPATH")

	for _, p := range panes {
		focused := "no"
		if p.Focused {
			focused = "yes"
		}

		//nolint:errcheck // writing to output; errors surface from Flush below
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			p.PaneID, p.WorkspaceID, p.PaneGroupID, focused, p.CurrentCommand, p.CurrentPath)
	}

	if err := tw.Flush(); err != nil {
		return fmt.Errorf("writing pane table: %w", err)
	}

	return nil
}
