package pane_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCloseCmd_ClosesPane(t *testing.T) {
	t.Parallel()

	sess := &stubSessionClient{}

	out, err := runPane(t, &stubManager{sess: sess},
		cmdClose, flagWorkspace, testWorkspaceID, flagPane, testPaneID)
	require.NoError(t, err)
	require.Empty(t, out, "a successful close says nothing")

	require.True(t, sess.closeCalled)
	require.Equal(t, testWorkspaceID, sess.closeReq.GetWorkspaceId())
	require.Equal(t, testPaneID, sess.closeReq.GetPaneId())
}

// A pane whose program already exited is the normal case for cleanup, and the
// contract makes ClosePane idempotent. The CLI must not add an error of its
// own on top of that.
func TestCloseCmd_AlreadyGonePaneSucceeds(t *testing.T) {
	t.Parallel()

	sess := &stubSessionClient{}

	_, err := runPane(t, &stubManager{sess: sess},
		cmdClose, flagWorkspace, testWorkspaceID, flagPane, missingPaneID)
	require.NoError(t, err)
	require.Equal(t, missingPaneID, sess.closeReq.GetPaneId())
}

func TestCloseCmd_MissingRequiredFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: caseNoFlags, args: []string{cmdClose}},
		{name: caseWorkspaceOnly, args: []string{cmdClose, flagWorkspace, testWorkspaceID}},
		{name: casePaneOnly, args: []string{cmdClose, flagPane, testPaneID}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sess := &stubSessionClient{}

			_, err := runPane(t, &stubManager{sess: sess}, tc.args...)
			require.Error(t, err)
			require.False(t, sess.closeCalled)
		})
	}
}

func TestCloseCmd_Errors(t *testing.T) {
	t.Parallel()

	t.Run("ClosePane fails", func(t *testing.T) {
		t.Parallel()

		mgr := &stubManager{sess: &stubSessionClient{closeErr: errPluginFailure}}

		_, err := runPane(t, mgr, cmdClose, flagWorkspace, testWorkspaceID, flagPane, testPaneID)
		require.ErrorIs(t, err, errPluginFailure)
	})

	t.Run(caseNoSession, func(t *testing.T) {
		t.Parallel()

		_, err := runPane(t, &stubManager{}, cmdClose, flagWorkspace, testWorkspaceID, flagPane, testPaneID)
		require.ErrorIs(t, err, errNoSessionPlugin)
	})
}
