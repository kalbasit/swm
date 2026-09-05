package pane_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"
)

const otherPaneID = "%7"

const otherPaneGroupID = "github.com/kalbasit/other"

// samplePanes is a two-pane stream spanning two pane groups, one of them
// focused and one reported by a provider that supplies no descriptive fields.
func samplePanes() []*pluginv1.Pane {
	return []*pluginv1.Pane{
		{
			PaneId:         testPaneID,
			PaneGroupId:    testPaneGroupID,
			WorkspaceId:    testWorkspaceID,
			Title:          testCommand,
			CurrentCommand: testCommand,
			CurrentPath:    testWorktree,
			Focused:        true,
		},
		{
			PaneId:      otherPaneID,
			PaneGroupId: otherPaneGroupID,
			WorkspaceId: testWorkspaceID,
		},
	}
}

func TestListCmd_JSON(t *testing.T) {
	t.Parallel()

	sess := &stubSessionClient{listPanes: samplePanes()}

	out, err := runPane(t, &stubManager{sess: sess}, cmdList, flagJSON)
	require.NoError(t, err)

	var got []map[string]any

	require.NoError(t, json.Unmarshal([]byte(out), &got))
	require.Len(t, got, 2)

	require.Equal(t, wantPaneJSON(
		testPaneID, testPaneGroupID, testWorkspaceID,
		testCommand, testCommand, testWorktree, true,
	), got[0])

	// A provider that cannot report a descriptive field leaves it at its zero
	// value; the key is still present so a consumer can index without an
	// existence check.
	require.Equal(t, wantPaneJSON(
		otherPaneID, otherPaneGroupID, testWorkspaceID,
		"", "", "", false,
	), got[1])

	// Unset filters mean every live workspace and every pane group in them.
	require.Empty(t, sess.listReq.GetWorkspaceId())
	require.Empty(t, sess.listReq.GetPaneGroupId())
}

func TestListCmd_ForwardsFilters(t *testing.T) {
	t.Parallel()

	sess := &stubSessionClient{}

	_, err := runPane(t, &stubManager{sess: sess},
		cmdList, flagWorkspace, testWorkspaceID, flagPaneGroup, testPaneGroupID)
	require.NoError(t, err)

	require.Equal(t, testWorkspaceID, sess.listReq.GetWorkspaceId())
	require.Equal(t, testPaneGroupID, sess.listReq.GetPaneGroupId())
}

func TestListCmd_Empty(t *testing.T) {
	t.Parallel()

	t.Run("json mode emits an empty array, never null", func(t *testing.T) {
		t.Parallel()

		out, err := runPane(t, &stubManager{sess: &stubSessionClient{}}, cmdList, flagJSON)
		require.NoError(t, err)
		require.Equal(t, "[]", strings.TrimSpace(out))
	})

	t.Run("table mode prints nothing, not even a header", func(t *testing.T) {
		t.Parallel()

		out, err := runPane(t, &stubManager{sess: &stubSessionClient{}}, cmdList)
		require.NoError(t, err)
		require.Empty(t, out)
	})
}

func TestListCmd_Table(t *testing.T) {
	t.Parallel()

	sess := &stubSessionClient{listPanes: samplePanes()}

	out, err := runPane(t, &stubManager{sess: sess}, cmdList)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	require.Len(t, lines, 3, "header plus one row per pane")
	require.Contains(t, lines[0], "PANE ID")
	require.Contains(t, lines[1], testPaneID)
	require.Contains(t, lines[1], "yes", "the focused pane must be visibly marked")
	require.Contains(t, lines[2], otherPaneID)
	require.Contains(t, lines[2], "no")
}

func TestListCmd_Errors(t *testing.T) {
	t.Parallel()

	t.Run("opening the stream fails", func(t *testing.T) {
		t.Parallel()

		_, err := runPane(t, &stubManager{sess: &stubSessionClient{listErr: errPluginFailure}}, cmdList)
		require.ErrorIs(t, err, errPluginFailure)
	})

	t.Run("stream fails mid-way", func(t *testing.T) {
		t.Parallel()

		sess := &stubSessionClient{listPanes: samplePanes(), recvErr: errPluginFailure}

		_, err := runPane(t, &stubManager{sess: sess}, cmdList, flagJSON)
		require.ErrorIs(t, err, errPluginFailure)
	})

	t.Run(caseNoSession, func(t *testing.T) {
		t.Parallel()

		_, err := runPane(t, &stubManager{}, cmdList)
		require.ErrorIs(t, err, errNoSessionPlugin)
	})
}
