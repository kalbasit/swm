package pane_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kalbasit/swm/cmd/swm/internal/cli/pane"
	"github.com/kalbasit/swm/cmd/swm/internal/exitcode"
)

// sendArgs is the flag prefix every `pane send` invocation in this file shares.
func sendArgs(extra ...string) []string {
	base := make([]string, 0, 5+len(extra))
	base = append(base, cmdSend, flagWorkspace, testWorkspaceID, flagPane, testPaneID)

	return append(base, extra...)
}

func TestSendCmd_DeliversText(t *testing.T) {
	t.Parallel()

	sess := &stubSessionClient{}

	out, err := runPane(t, &stubManager{sess: sess}, sendArgs(testText)...)
	require.NoError(t, err)
	require.Empty(t, out, "a successful send says nothing")

	require.Equal(t, testWorkspaceID, sess.sendReq.GetWorkspaceId())
	require.Equal(t, testPaneID, sess.sendReq.GetPaneId())
	require.Equal(t, testText, sess.sendReq.GetText())
	require.False(t, sess.sendReq.GetSubmit())
	require.False(t, sess.sendReq.GetAllowFocused(),
		"the CLI must never set allow_focused on the caller's behalf")
	require.Zero(t, sess.sendReq.GetDelayMs())
}

func TestSendCmd_TextIsNotInterpreted(t *testing.T) {
	t.Parallel()

	sess := &stubSessionClient{}

	// "Enter" is five characters of text, not the Enter key. The CLI passes it
	// through untouched and leaves that guarantee to the plugin.
	_, err := runPane(t, &stubManager{sess: sess}, sendArgs("Enter")...)
	require.NoError(t, err)

	require.Equal(t, "Enter", sess.sendReq.GetText())
	require.False(t, sess.sendReq.GetSubmit())
}

func TestSendCmd_SubmitAndDelay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantText   string
		wantSubmit bool
		wantDelay  int32
	}{
		{
			name:       "submit key only",
			args:       []string{flagSubmit},
			wantText:   "",
			wantSubmit: true,
		},
		{
			name:       "text then submit",
			args:       []string{flagSubmit, testText},
			wantText:   testText,
			wantSubmit: true,
		},
		{
			name:       "delayed submit key",
			args:       []string{flagSubmit, "--delay-ms", "250"},
			wantSubmit: true,
			wantDelay:  250,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sess := &stubSessionClient{}

			_, err := runPane(t, &stubManager{sess: sess}, sendArgs(tc.args...)...)
			require.NoError(t, err)

			require.Equal(t, tc.wantText, sess.sendReq.GetText())
			require.Equal(t, tc.wantSubmit, sess.sendReq.GetSubmit())
			require.Equal(t, tc.wantDelay, sess.sendReq.GetDelayMs())
		})
	}
}

func TestSendCmd_NothingToSend(t *testing.T) {
	t.Parallel()

	sess := &stubSessionClient{}

	_, err := runPane(t, &stubManager{sess: sess}, sendArgs()...)
	require.Error(t, err)
	require.Contains(t, err.Error(), flagSubmit)
	require.Nil(t, sess.sendReq, "no plugin call for a delivery that carries nothing")
	require.Equal(t, exitcode.Failure, exitcode.From(err),
		"a local argument error is an ordinary failure, not the focused refusal")
}

func TestSendCmd_MissingRequiredFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: caseNoFlags, args: []string{cmdSend, testText}},
		{name: caseWorkspaceOnly, args: []string{cmdSend, flagWorkspace, testWorkspaceID, testText}},
		{name: casePaneOnly, args: []string{cmdSend, flagPane, testPaneID, testText}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sess := &stubSessionClient{}

			_, err := runPane(t, &stubManager{sess: sess}, tc.args...)
			require.Error(t, err)
			require.Nil(t, sess.sendReq)
		})
	}
}

func TestSendCmd_FocusedPaneRefusal(t *testing.T) {
	t.Parallel()

	sess := &stubSessionClient{sendErr: errFocused}

	_, err := runPane(t, &stubManager{sess: sess}, sendArgs(testText)...)
	require.Error(t, err)

	// A caller has to be able to branch on this without reading the message,
	// because it is the one failure whose right answer is to back off and retry.
	require.Equal(t, pane.ExitFocusedPane, exitcode.From(err))
	require.Contains(t, err.Error(), "--allow-focused")
	require.Contains(t, err.Error(), testPaneID)
}

func TestSendCmd_AllowFocused(t *testing.T) {
	t.Parallel()

	sess := &stubSessionClient{}

	_, err := runPane(t, &stubManager{sess: sess}, sendArgs("--allow-focused", testText)...)
	require.NoError(t, err)
	require.True(t, sess.sendReq.GetAllowFocused())
}

func TestSendCmd_OtherFailuresStayOnTheGenericStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
	}{
		{name: "pane not found", err: errPaneNotFound},
		{name: "plugin failure with no status", err: errPluginFailure},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mgr := &stubManager{sess: &stubSessionClient{sendErr: tc.err}}

			_, err := runPane(t, mgr, sendArgs(testText)...)
			require.Error(t, err)
			require.Equal(t, exitcode.Failure, exitcode.From(err))
			require.NotEqual(t, pane.ExitFocusedPane, exitcode.From(err))
		})
	}
}

func TestSendCmd_NoSessionPlugin(t *testing.T) {
	t.Parallel()

	_, err := runPane(t, &stubManager{}, sendArgs(testText)...)
	require.ErrorIs(t, err, errNoSessionPlugin)
	require.Equal(t, exitcode.Failure, exitcode.From(err))
}
