package pane_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"
)

// openArgs is the flag prefix every `pane open` invocation in this file shares.
func openArgs(extra ...string) []string {
	base := make([]string, 0, 5+len(extra))
	base = append(base, cmdOpen, flagWorkspace, testWorkspaceID, flagPaneGroup, testPaneGroupID)

	return append(base, extra...)
}

func TestOpenCmd_PrintsPaneID(t *testing.T) {
	t.Parallel()

	sess := &stubSessionClient{}

	out, err := runPane(t, &stubManager{sess: sess},
		openArgs("--", testCommand, "main.go")...)
	require.NoError(t, err)

	// The whole point of the default output is `pane=$(swm pane open ...)`, so
	// it must be the identifier and nothing else.
	require.Equal(t, testPaneID+"\n", out)

	require.Equal(t, testWorkspaceID, sess.openReq.GetWorkspaceId())
	require.Equal(t, testPaneGroupID, sess.openReq.GetPaneGroupId())
	require.Equal(t, []string{testCommand, "main.go"}, sess.openReq.GetArgv())
}

func TestOpenCmd_Argv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "no argv opens the provider default shell",
			args: nil,
			want: []string{},
		},
		{
			name: "element containing spaces is not re-split",
			args: []string{"--", "sh", "-c", "echo hello world"},
			want: []string{"sh", "-c", "echo hello world"},
		},
		{
			name: "leading dash after the separator is not parsed as a flag",
			args: []string{"--", testCommand, "-u", "NONE"},
			want: []string{testCommand, "-u", "NONE"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sess := &stubSessionClient{}

			_, err := runPane(t, &stubManager{sess: sess}, openArgs(tc.args...)...)
			require.NoError(t, err)
			require.Equal(t, tc.want, sess.openReq.GetArgv())
		})
	}
}

func TestOpenCmd_CwdAndEnv(t *testing.T) {
	t.Parallel()

	sess := &stubSessionClient{}

	_, err := runPane(t, &stubManager{sess: sess}, openArgs(
		"--cwd", testWorktree,
		"--env", "FOO=bar",
		"--env", "EQ=a=b",
		"--env", "EMPTY=",
	)...)
	require.NoError(t, err)

	require.Equal(t, testWorktree, sess.openReq.GetCwd())
	require.Equal(t, map[string]string{"FOO": "bar", "EQ": "a=b", "EMPTY": ""}, sess.openReq.GetEnv())
}

func TestOpenCmd_MalformedEnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		entry string
	}{
		{name: "no separator", entry: "FOO"},
		{name: "empty key", entry: "=value"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sess := &stubSessionClient{}

			_, err := runPane(t, &stubManager{sess: sess}, openArgs("--env", tc.entry)...)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.entry)
			require.Nil(t, sess.openReq, "the plugin must not be called with arguments known to be wrong")
		})
	}
}

func TestOpenCmd_JSON(t *testing.T) {
	t.Parallel()

	sess := &stubSessionClient{openPane: &pluginv1.Pane{
		PaneId:         testPaneID,
		PaneGroupId:    testPaneGroupID,
		WorkspaceId:    testWorkspaceID,
		Title:          testCommand,
		CurrentCommand: testCommand,
		CurrentPath:    testWorktree,
		Focused:        true,
	}}

	out, err := runPane(t, &stubManager{sess: sess}, openArgs(flagJSON)...)
	require.NoError(t, err)

	var got map[string]any

	require.NoError(t, json.Unmarshal([]byte(out), &got))
	require.Equal(t, wantPaneJSON(
		testPaneID, testPaneGroupID, testWorkspaceID,
		testCommand, testCommand, testWorktree, true,
	), got)
}

func TestOpenCmd_MissingRequiredFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "no flags at all", args: []string{cmdOpen}},
		{name: caseWorkspaceOnly, args: []string{cmdOpen, flagWorkspace, testWorkspaceID}},
		{name: "pane group only", args: []string{cmdOpen, flagPaneGroup, testPaneGroupID}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sess := &stubSessionClient{}

			_, err := runPane(t, &stubManager{sess: sess}, tc.args...)
			require.Error(t, err)
			require.Nil(t, sess.openReq)
		})
	}
}

func TestOpenCmd_PluginErrors(t *testing.T) {
	t.Parallel()

	t.Run("OpenPane fails", func(t *testing.T) {
		t.Parallel()

		mgr := &stubManager{sess: &stubSessionClient{openErr: errPluginFailure}}

		_, err := runPane(t, mgr, openArgs()...)
		require.ErrorIs(t, err, errPluginFailure)
	})

	t.Run(caseNoSession, func(t *testing.T) {
		t.Parallel()

		_, err := runPane(t, &stubManager{}, openArgs()...)
		require.ErrorIs(t, err, errNoSessionPlugin)
	})

	t.Run("session capability resolves to the wrong type", func(t *testing.T) {
		t.Parallel()

		_, err := runPane(t, &wrongPluginManager{}, openArgs()...)
		require.Error(t, err)
		require.Contains(t, err.Error(), "SessionClient")
	})
}
