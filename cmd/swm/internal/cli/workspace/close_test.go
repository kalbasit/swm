package workspace_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"
	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/cmd/swm/internal/cli/workspace"
)

const (
	testCloseWorkspaceID = "sock-feat-x"
	testCompletionStoryA = "feat-a"
	testCompletionStoryB = "feat-b"
)

// stubCloseSession is a SessionClient for close command tests with configurable
// ListWorkspaces and CloseWorkspace behavior.
type stubCloseSession struct {
	workspaces       []*pluginv1.Workspace
	listErr          error
	closeWorkspaceID string // set by CloseWorkspace call
	closeErr         error
}

func (s *stubCloseSession) CloseWorkspace(
	_ context.Context,
	req *pluginv1.CloseWorkspaceRequest,
	_ ...grpc.CallOption,
) (*pluginv1.Empty, error) {
	s.closeWorkspaceID = req.GetWorkspaceId()

	return &pluginv1.Empty{}, s.closeErr
}

func (s *stubCloseSession) CurrentContext(
	context.Context, *pluginv1.Empty, ...grpc.CallOption,
) (*pluginv1.CurrentContextResponse, error) {
	panic("stub")
}

func (s *stubCloseSession) Info(
	context.Context, *pluginv1.Empty, ...grpc.CallOption,
) (*pluginv1.SessionInfo, error) {
	panic("stub")
}

func (s *stubCloseSession) IsInsideWorkspace(
	context.Context, *pluginv1.Empty, ...grpc.CallOption,
) (*pluginv1.BoolValue, error) {
	panic("stub")
}

func (s *stubCloseSession) ListWorkspaces(
	_ context.Context, _ *pluginv1.Empty, _ ...grpc.CallOption,
) (grpc.ServerStreamingClient[pluginv1.Workspace], error) {
	if s.listErr != nil {
		return nil, s.listErr
	}

	return &staticWorkspaceStream{workspaces: s.workspaces}, nil
}

func (s *stubCloseSession) OpenPaneGroup(
	context.Context, *pluginv1.OpenPaneGroupRequest, ...grpc.CallOption,
) (*pluginv1.PaneGroup, error) {
	panic("stub")
}

func (s *stubCloseSession) OpenWorkspace(
	context.Context, *pluginv1.OpenWorkspaceRequest, ...grpc.CallOption,
) (*pluginv1.Workspace, error) {
	panic("stub")
}

func (s *stubCloseSession) SwitchTo(
	context.Context, *pluginv1.SwitchToRequest, ...grpc.CallOption,
) (*pluginv1.SwitchToResponse, error) {
	panic("stub")
}

var _ pluginv1.SessionClient = (*stubCloseSession)(nil)

// staticWorkspaceStream streams a fixed slice of workspaces, then EOF.
type staticWorkspaceStream struct {
	workspaces []*pluginv1.Workspace
	pos        int
}

func (s *staticWorkspaceStream) CloseSend() error             { return nil }
func (s *staticWorkspaceStream) Context() context.Context     { return context.Background() }
func (s *staticWorkspaceStream) Header() (metadata.MD, error) { panic("stub") }

func (s *staticWorkspaceStream) Recv() (*pluginv1.Workspace, error) {
	if s.pos >= len(s.workspaces) {
		return nil, io.EOF
	}

	ws := s.workspaces[s.pos]
	s.pos++

	return ws, nil
}

func (s *staticWorkspaceStream) RecvMsg(any) error    { panic("stub") }
func (s *staticWorkspaceStream) SendMsg(any) error    { panic("stub") }
func (s *staticWorkspaceStream) Trailer() metadata.MD { panic("stub") }

func TestCloseCmd_ClosesRunningWorkspace(t *testing.T) {
	t.Parallel()

	sess := &stubCloseSession{
		workspaces: []*pluginv1.Workspace{
			{WorkspaceId: testCloseWorkspaceID, StoryName: testStoryName},
		},
	}
	mgr := &stubMgr{sess: sess}
	store := &stubStore{}

	cmd := workspace.NewCloseCmd(store, mgr)
	out := &strings.Builder{}
	cmd.SetOut(out)
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute())
	require.Equal(t, testCloseWorkspaceID, sess.closeWorkspaceID)
	require.Contains(t, out.String(), `closed workspace for story "feat-x"`)
}

func TestCloseCmd_NoActiveWorkspace_Idempotent(t *testing.T) {
	t.Parallel()

	sess := &stubCloseSession{workspaces: nil}
	mgr := &stubMgr{sess: sess}
	store := &stubStore{}

	cmd := workspace.NewCloseCmd(store, mgr)
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute())
	require.Empty(t, sess.closeWorkspaceID)
}

func TestCloseCmd_SWMStoryFallback(t *testing.T) {
	t.Setenv("SWM_STORY", testStoryName)

	sess := &stubCloseSession{
		workspaces: []*pluginv1.Workspace{
			{WorkspaceId: testCloseWorkspaceID, StoryName: testStoryName},
		},
	}
	mgr := &stubMgr{sess: sess}
	store := &stubStore{}

	cmd := workspace.NewCloseCmd(store, mgr)
	cmd.SetArgs([]string{})

	require.NoError(t, cmd.Execute())
	require.Equal(t, testCloseWorkspaceID, sess.closeWorkspaceID)
}

func TestCloseCmd_NoArg_NoEnv_Error(t *testing.T) {
	t.Setenv("SWM_STORY", "")

	mgr := &stubMgr{}
	store := &stubStore{}

	cmd := workspace.NewCloseCmd(store, mgr)
	cmd.SetArgs([]string{})

	require.Error(t, cmd.Execute())
}

func TestCloseCmd_ArgOverridesEnv(t *testing.T) {
	t.Setenv("SWM_STORY", "other-story")

	sess := &stubCloseSession{
		workspaces: []*pluginv1.Workspace{
			{WorkspaceId: testCloseWorkspaceID, StoryName: testStoryName},
		},
	}
	mgr := &stubMgr{sess: sess}
	store := &stubStore{}

	cmd := workspace.NewCloseCmd(store, mgr)
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute())
	require.Equal(t, testCloseWorkspaceID, sess.closeWorkspaceID)
}

func TestCloseCmd_SessionPluginAbsent(t *testing.T) {
	t.Parallel()

	mgr := &stubMgr{} // no sess → Get("session") returns errNoPlugin
	store := &stubStore{}

	cmd := workspace.NewCloseCmd(store, mgr)
	cmd.SetArgs([]string{testStoryName})

	require.Error(t, cmd.Execute())
}

func TestCloseCmd_ListWorkspacesError(t *testing.T) {
	t.Parallel()

	sess := &stubCloseSession{listErr: errNoPlugin}
	mgr := &stubMgr{sess: sess}
	store := &stubStore{}

	cmd := workspace.NewCloseCmd(store, mgr)
	cmd.SetArgs([]string{testStoryName})

	err := cmd.Execute()
	require.Error(t, err)
	require.ErrorIs(t, err, errNoPlugin)
}

func TestCloseCmd_ShellCompletion_ListsStories(t *testing.T) {
	t.Parallel()

	store := &stubStore{
		listStories: []*coreStory.Story{
			{Name: testCompletionStoryA},
			{Name: testCompletionStoryB},
		},
	}
	mgr := &stubMgr{}

	cmd := workspace.NewCloseCmd(store, mgr)

	completions, directive := cmd.ValidArgsFunction(cmd, []string{}, "")
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	require.ElementsMatch(t, []string{testCompletionStoryA, testCompletionStoryB}, completions)
}

func TestCloseCmd_ShellCompletion_NoCompletionAfterFirstArg(t *testing.T) {
	t.Parallel()

	store := &stubStore{}
	mgr := &stubMgr{}

	cmd := workspace.NewCloseCmd(store, mgr)

	_, directive := cmd.ValidArgsFunction(cmd, []string{"already-provided"}, "")
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}

// badTypeMgr returns a wrong type for the session plugin to exercise the type
// assertion failure path in NewCloseCmd.
type badTypeMgr struct{}

func (b *badTypeMgr) Close() error                                 { return nil }
func (b *badTypeMgr) Get(_ context.Context, _ string) (any, error) { return struct{}{}, nil }
func (b *badTypeMgr) Warm(_ context.Context, _ ...string) error    { return nil }

func TestCloseCmd_WrongSessionPluginType_ErrorIncludesType(t *testing.T) {
	t.Parallel()

	store := &stubStore{}
	cmd := workspace.NewCloseCmd(store, &badTypeMgr{})
	cmd.SetArgs([]string{testStoryName})

	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "struct {}")
}

func TestCloseCmd_CloseWorkspaceError_ErrorIncludesContext(t *testing.T) {
	t.Parallel()

	sess := &stubCloseSession{
		workspaces: []*pluginv1.Workspace{
			{WorkspaceId: testCloseWorkspaceID, StoryName: testStoryName},
		},
		closeErr: errFakeClose,
	}
	mgr := &stubMgr{sess: sess}
	store := &stubStore{}

	cmd := workspace.NewCloseCmd(store, mgr)
	cmd.SetArgs([]string{testStoryName})

	err := cmd.Execute()
	require.Error(t, err)
	require.ErrorIs(t, err, errFakeClose)
	require.Contains(t, err.Error(), testCloseWorkspaceID)
	require.Contains(t, err.Error(), testStoryName)
}

func TestCloseCmd_ShellCompletion_NilStoriesSkipped(t *testing.T) {
	t.Parallel()

	store := &stubStore{
		listStories: []*coreStory.Story{
			{Name: testCompletionStoryA},
			nil,
			{Name: testCompletionStoryB},
		},
	}
	mgr := &stubMgr{}

	cmd := workspace.NewCloseCmd(store, mgr)

	completions, directive := cmd.ValidArgsFunction(cmd, []string{}, "")
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	require.ElementsMatch(t, []string{testCompletionStoryA, testCompletionStoryB}, completions)
}
