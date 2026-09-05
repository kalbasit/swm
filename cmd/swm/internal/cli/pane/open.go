package pane

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"
)

// errMalformedEnv is returned for an --env value that is not KEY=VALUE.
var errMalformedEnv = errors.New("malformed --env entry")

// NewOpenCmd returns the `swm pane open` command.
func NewOpenCmd(mgr PluginManager) *cobra.Command {
	var (
		workspaceID string
		paneGroupID string
		cwd         string
		envEntries  []string
		asJSON      bool
	)

	cmd := &cobra.Command{
		Use:   "open [flags] [-- <command> [args...]]",
		Short: "Open a new pane running a program",
		Long: "Start a program in a new pane inside a pane group.\n\n" +
			"Everything after -- is the program and its arguments, already split: an " +
			"argument containing spaces is passed through intact and is never " +
			"re-split. With no program the plugin starts its default shell.\n\n" +
			"By default the new pane's id is printed on stdout and nothing else, so it " +
			"can be captured directly: pane=$(swm pane open -w ... -g ...).",
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse before resolving the plugin: there is no point starting a
			// plugin process only to be told the arguments were wrong.
			env, err := parseEnv(envEntries)
			if err != nil {
				return err
			}

			ctx := cmd.Context()

			sess, err := sessionClient(ctx, mgr)
			if err != nil {
				return err
			}

			pane, err := sess.OpenPane(ctx, &pluginv1.OpenPaneRequest{
				WorkspaceId: workspaceID,
				PaneGroupId: paneGroupID,
				Argv:        args,
				Cwd:         cwd,
				Env:         env,
			})
			if err != nil {
				return fmt.Errorf("opening pane in pane group %q: %w", paneGroupID, err)
			}

			if asJSON {
				return writeJSON(cmd.OutOrStdout(), newPaneView(pane))
			}

			//nolint:errcheck // writing to output; errors are non-actionable
			fmt.Fprintln(cmd.OutOrStdout(), pane.GetPaneId())

			return nil
		},
	}

	cmd.Flags().StringVarP(&workspaceID, "workspace", "w", "",
		"workspace id the pane group lives in (required)")
	cmd.Flags().StringVarP(&paneGroupID, "pane-group", "g", "",
		"pane group to open the pane in (required)")
	cmd.Flags().StringVar(&cwd, "cwd", "", "starting directory for the new pane")
	cmd.Flags().StringArrayVar(&envEntries, "env", nil,
		"environment entry KEY=VALUE for the new pane (repeatable)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print the pane as a JSON object")

	markRequired(cmd, "workspace", "pane-group")

	return cmd
}

// parseEnv turns repeated KEY=VALUE flag values into the request's env map.
//
// Only the first "=" separates, so a value may itself contain them — an
// entry like PATH=/a:/b=c is a value, not an error. An entry with no separator
// or an empty key is rejected rather than guessed at.
func parseEnv(entries []string) (map[string]string, error) {
	// Always non-nil: an empty map marshals the same as an absent one, and
	// returning a nil map alongside a nil error would be two ways of saying
	// the same thing.
	env := make(map[string]string, len(entries))

	for _, entry := range entries {
		key, value, found := strings.Cut(entry, "=")
		if !found || key == "" {
			return nil, fmt.Errorf("%w %q: expected KEY=VALUE", errMalformedEnv, entry)
		}

		env[key] = value
	}

	return env, nil
}
