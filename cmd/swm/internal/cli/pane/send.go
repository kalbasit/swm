package pane

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/cmd/swm/internal/exitcode"
)

// ExitFocusedPane is the process exit status `swm pane send` uses when the
// session plugin refuses to deliver into a pane it reports as focused.
//
// It has a status of its own because it is the one failure whose correct
// response differs: a caller should back off and retry, not give up. Making
// that distinguishable by exit status means no caller has to match on the text
// of an error message.
//
// 3 rather than 2, which is conventionally a usage error.
const ExitFocusedPane = 3

// errNothingToSend is returned when a send would deliver nothing at all.
var errNothingToSend = errors.New("nothing to send: pass <text>, --submit, or both")

// NewSendCmd returns the `swm pane send` command.
func NewSendCmd(mgr PluginManager) *cobra.Command {
	var (
		workspaceID  string
		paneID       string
		submit       bool
		delayMS      int32
		allowFocused bool
	)

	cmd := &cobra.Command{
		Use:   "send [flags] [<text>]",
		Short: "Deliver text to a pane",
		Long: "Deliver text to a pane exactly as if it had been typed there.\n\n" +
			"The text is literal: no part of it is read as a key name, so \"Enter\" " +
			"arrives as five characters rather than the Enter key. Use --submit to " +
			"follow it with the submit key.\n\n" +
			"Delivery into a pane an attached client is focused on is refused, because " +
			"the text would be consumed by whatever that person is in the middle of " +
			"answering. The command then exits " + strconv.Itoa(ExitFocusedPane) + ". " +
			"--allow-focused sends anyway; a caller that sets it owns the consequences, " +
			"and the check is racy in any case.",
		Args:    cobra.MaximumNArgs(1),
		PreRunE: warmSession(mgr),
		RunE: func(cmd *cobra.Command, args []string) error {
			var text string
			if len(args) == 1 {
				text = args[0]
			}

			// The plugin rejects this too, but rejecting it here names the flag
			// the caller actually typed and avoids starting a plugin process
			// only to be told the arguments were wrong.
			if text == "" && !submit {
				return errNothingToSend
			}

			ctx := cmd.Context()

			sess, err := sessionClient(ctx, mgr)
			if err != nil {
				return err
			}

			if _, err := sess.SendText(ctx, &pluginv1.SendTextRequest{
				WorkspaceId:  workspaceID,
				PaneId:       paneID,
				Text:         text,
				Submit:       submit,
				DelayMs:      delayMS,
				AllowFocused: allowFocused,
			}); err != nil {
				return sendFailure(paneID, err)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&workspaceID, "workspace", "w", "",
		"workspace id the pane belongs to (required)")
	cmd.Flags().StringVarP(&paneID, "pane", "p", "", "pane id to deliver to (required)")
	cmd.Flags().BoolVar(&submit, "submit", false,
		"follow the text with the provider's submit key")
	cmd.Flags().Int32Var(&delayMS, "delay-ms", 0,
		"wait this many milliseconds before delivering")
	cmd.Flags().BoolVar(&allowFocused, "allow-focused", false,
		"deliver even into a pane an attached client is focused on")

	markRequired(cmd, "workspace", "pane")

	return cmd
}

// sendFailure maps a SendText failure onto the error the command returns.
//
// The Session contract assigns FAILED_PRECONDITION on SendText to exactly one
// condition — the pane is focused and allow_focused was not set — so the status
// code is matched rather than the message. A provider using that code for
// anything else on this RPC is out of conformance, and matching on English
// would be worse in every direction.
func sendFailure(paneID string, err error) error {
	if status.Code(err) == codes.FailedPrecondition {
		return exitcode.Wrap(ExitFocusedPane, fmt.Errorf(
			"pane %s is focused: delivering text would type into a pane someone is using; "+
				"pass --allow-focused to send anyway: %w", paneID, err))
	}

	return fmt.Errorf("sending text to pane %s: %w", paneID, err)
}
