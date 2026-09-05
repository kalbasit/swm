package pane

import (
	"fmt"

	"github.com/spf13/cobra"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"
)

// NewCloseCmd returns the `swm pane close` command.
func NewCloseCmd(mgr PluginManager) *cobra.Command {
	var (
		workspaceID string
		paneID      string
	)

	cmd := &cobra.Command{
		Use:   "close",
		Short: "Close a pane",
		Long: "Terminate a pane.\n\n" +
			"Closing a pane that no longer exists is not an error: cleaning up after a " +
			"program that already exited on its own is the normal case.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			sess, err := sessionClient(ctx, mgr)
			if err != nil {
				return err
			}

			if _, err := sess.ClosePane(ctx, &pluginv1.ClosePaneRequest{
				WorkspaceId: workspaceID,
				PaneId:      paneID,
			}); err != nil {
				return fmt.Errorf("closing pane %s: %w", paneID, err)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&workspaceID, "workspace", "w", "",
		"workspace id the pane belongs to (required)")
	cmd.Flags().StringVarP(&paneID, "pane", "p", "", "pane id to close (required)")

	markRequired(cmd, "workspace", "pane")

	return cmd
}
