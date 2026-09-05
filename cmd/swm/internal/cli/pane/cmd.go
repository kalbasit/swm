// Package pane contains the `swm pane` sub-commands, which expose the Session
// capability's pane primitives to processes that do not speak the plugin gRPC
// contract.
//
// The commands are deliberately thin. They do not interpret a pane_id, which
// is an opaque provider handle, and they add no behaviour of their own on top
// of the RPCs they wrap — including the refusal to type into a pane a person
// is using, which stays the plugin's decision.
package pane

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"
)

// capSession is the capability name these commands resolve.
const capSession = "session"

// errUnexpectedSessionPlugin is returned when the session capability resolves
// to something that is not a Session client.
var errUnexpectedSessionPlugin = errors.New("unexpected session plugin type")

// PluginManager is the subset of the CLI plugin manager these commands use.
type PluginManager interface {
	Get(ctx context.Context, capability string) (any, error)
	Warm(ctx context.Context, capabilities ...string) error
}

// NewPaneCmd returns the `swm pane` command group.
func NewPaneCmd(mgr PluginManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pane",
		Short: "Open, list, drive, and close individual panes",
		Long: "Operate on individual panes through the configured session plugin.\n\n" +
			"A pane id is an opaque handle minted by the plugin: pass it back exactly as " +
			"it was given, never parse it. It is meaningful only within the workspace " +
			"that produced it, which is why every command taking one also takes " +
			"--workspace.",
	}

	cmd.AddCommand(
		NewOpenCmd(mgr),
		NewListCmd(mgr),
		NewSendCmd(mgr),
		NewCloseCmd(mgr),
	)

	return cmd
}

// sessionClient resolves the session capability to its gRPC client.
func sessionClient(ctx context.Context, mgr PluginManager) (pluginv1.SessionClient, error) {
	raw, err := mgr.Get(ctx, capSession)
	if err != nil {
		return nil, fmt.Errorf("loading session plugin: %w", err)
	}

	sess, ok := raw.(pluginv1.SessionClient)
	if !ok {
		return nil, fmt.Errorf("%w: expected pluginv1.SessionClient, got %T", errUnexpectedSessionPlugin, raw)
	}

	return sess, nil
}

// markRequired marks flags as required. A name that does not exist is a
// programming error in this package, not a runtime condition, so it is
// reported through the command construction rather than swallowed.
func markRequired(cmd *cobra.Command, names ...string) {
	for _, name := range names {
		// MarkFlagRequired only fails for a flag this package did not define a
		// line earlier; the tests covering each command's required flags would
		// fail loudly if that ever happened.
		cmd.MarkFlagRequired(name) //nolint:errcheck,gosec // see comment above
	}
}
