package workspace_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"
	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/cmd/swm/internal/cli/workspace"
	"github.com/kalbasit/swm/cmd/swm/internal/config"
	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
	"github.com/kalbasit/swm/cmd/swm/internal/hookexec"
)

const (
	testCodeRoot     = "/code"
	testDefaultStory = "_default"
	testStoryName    = "feat-x"
	testHost         = "github.com"
	testOwner        = "kalbasit"
	testSegment      = "swm"
)

var errNoPlugin = errors.New("no plugin")

func TestOpenCmd_WithPositionalArg(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{Name: testStoryName}}
	sess := &stubSess{}
	mgr := &stubMgr{sess: sess}
	resolver := layout.NewResolver(testCodeRoot)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute())
	require.Equal(t, testStoryName, sess.lastOpenReq.GetStoryName())
}

func TestOpenCmd_PositionalArgOverridesEnv(t *testing.T) {
	t.Setenv("SWM_STORY", "env-story")

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{Name: testStoryName}}
	sess := &stubSess{}
	mgr := &stubMgr{sess: sess}
	resolver := layout.NewResolver(testCodeRoot)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute())
	require.Equal(t, testStoryName, sess.lastOpenReq.GetStoryName())
}

func TestOpenCmd_DefaultStory(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{Name: testDefaultStory}}
	sess := &stubSess{}
	mgr := &stubMgr{sess: sess}
	resolver := layout.NewResolver(testCodeRoot)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{})

	require.NoError(t, cmd.Execute())
	require.Equal(t, testDefaultStory, sess.lastOpenReq.GetStoryName())
}

func TestOpenCmd_StoryNotFound(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getErr: coreStory.ErrStoryNotFound}
	mgr := &stubMgr{}
	resolver := layout.NewResolver(testCodeRoot)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{"nonexistent"})

	require.Error(t, cmd.Execute())
}

func TestOpenCmd_NoProjects(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{Name: testStoryName, Projects: nil}}
	sess := &stubSess{}
	mgr := &stubMgr{sess: sess}
	resolver := layout.NewResolver(testCodeRoot)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute())
	require.Empty(t, sess.lastOpenReq.GetWorktreePaths())
}

func TestOpenCmd_WithPicker_ProjectAlreadyAttached(t *testing.T) {
	t.Parallel()

	const selectedKey = "github.com/kalbasit/swm"

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{
		Name: testStoryName,
		Projects: []coreStory.Project{
			{Host: testHost, Segments: []string{testOwner, testSegment}},
		},
	}}
	sess := &stubSess{}
	vcs := &stubVCS{}
	picker := &stubPickerClient{selectedKey: selectedKey}
	mgr := &stubMgr{sess: sess, vcs: vcs, picker: picker}
	resolver := layout.NewResolver(testCodeRoot)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute())

	// Already attached — no CreateWorktree call.
	require.False(t, vcs.createCalled)

	// Pane group was opened.
	require.NotNil(t, sess.lastPaneGroupReq)
	require.Equal(t, selectedKey, sess.lastPaneGroupReq.GetProjectId().GetHost()+"/"+
		"kalbasit/"+testSegment)
}

func TestOpenCmd_WithPicker_ProjectNotAttached(t *testing.T) {
	t.Parallel()

	const selectedKey = "github.com/kalbasit/dotfiles"

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{
		Name:       testStoryName,
		BranchName: "feat/feat-x",
	}}
	sess := &stubSess{}
	vcs := &stubVCS{}
	picker := &stubPickerClient{selectedKey: selectedKey}
	mgr := &stubMgr{sess: sess, vcs: vcs, picker: picker}
	resolver := layout.NewResolver(testCodeRoot)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute())

	// Not attached — CreateWorktree must be called.
	require.True(t, vcs.createCalled)

	// Story store updated.
	require.True(t, store.updateCalled)

	// Pane group was opened.
	require.NotNil(t, sess.lastPaneGroupReq)
}

func TestOpenCmd_WithPicker_Cancelled(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{Name: testStoryName}}
	sess := &stubSess{}
	vcs := &stubVCS{}
	// Picker returns Aborted (user pressed Escape).
	picker := &stubPickerClient{cancelOnRecv: true}
	mgr := &stubMgr{sess: sess, vcs: vcs, picker: picker}
	resolver := layout.NewResolver(testCodeRoot)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName})

	// Cancellation is not an error.
	require.NoError(t, cmd.Execute())
	require.Nil(t, sess.lastPaneGroupReq)
}

func TestOpenCmd_WithPicker_FailedPrecondition_FallsBack(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{
		Name: testStoryName,
		Projects: []coreStory.Project{
			{Host: testHost, Segments: []string{testOwner, testSegment}},
		},
	}}
	sess := &stubSess{}
	// Picker returns FailedPrecondition (e.g. no TTY available).
	picker := &stubPickerClient{pickErr: status.Error(codes.FailedPrecondition, "no tty")}
	mgr := &stubMgr{sess: sess, picker: picker}
	resolver := layout.NewResolver(testCodeRoot)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName})

	// Should succeed by falling back to Phase-1 behavior.
	require.NoError(t, cmd.Execute())

	// Phase-1 path: OpenWorkspace was called (not OpenPaneGroup).
	require.NotNil(t, sess.lastOpenReq, "expected OpenWorkspace to be called as fallback")
	require.Nil(t, sess.lastPaneGroupReq, "expected OpenPaneGroup NOT to be called")
}

// TestOpenCmd_WithPicker_RecvFailedPrecondition_FallsBack covers the case where the
// picker's stream.Recv() returns FailedPrecondition (e.g. /dev/tty unavailable inside
// the handler, after all candidates have been received). The host must still fall back
// to openAllAttached, not surface the gRPC error to the caller.
func TestOpenCmd_WithPicker_RecvFailedPrecondition_FallsBack(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{
		Name: testStoryName,
		Projects: []coreStory.Project{
			{Host: testHost, Segments: []string{testOwner, testSegment}},
		},
	}}
	sess := &stubSess{}
	// Picker stream opens fine but Recv() returns FailedPrecondition (no TTY).
	picker := &stubPickerClient{recvErr: status.Error(codes.FailedPrecondition, "no tty")}
	mgr := &stubMgr{sess: sess, picker: picker}
	resolver := layout.NewResolver(testCodeRoot)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute())
	require.NotNil(t, sess.lastOpenReq, "expected OpenWorkspace to be called as fallback")
	require.Nil(t, sess.lastPaneGroupReq, "expected OpenPaneGroup NOT to be called")
}

func TestOpenCmd_WithPicker_InvalidKey_EmptyHost(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{Name: testStoryName}}
	sess := &stubSess{}
	picker := &stubPickerClient{selectedKey: "/seg1"} // empty host
	mgr := &stubMgr{sess: sess, picker: picker}
	resolver := layout.NewResolver(testCodeRoot)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName})

	err := cmd.Execute()
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid project key")
}

func TestOpenCmd_WithPicker_InvalidKey_EmptySegments(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{Name: testStoryName}}
	sess := &stubSess{}
	picker := &stubPickerClient{selectedKey: "github.com/"} // empty segments
	mgr := &stubMgr{sess: sess, picker: picker}
	resolver := layout.NewResolver(testCodeRoot)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName})

	err := cmd.Execute()
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid project key")
}

func TestOpenCmd_WithPicker_InvalidKey_EmptySegmentPart(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{Name: testStoryName}}
	sess := &stubSess{}
	picker := &stubPickerClient{selectedKey: "github.com/seg1//seg3"} // empty middle segment
	mgr := &stubMgr{sess: sess, picker: picker}
	resolver := layout.NewResolver(testCodeRoot)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName})

	err := cmd.Execute()
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid project key")
}

func TestOpenCmd_WithPicker_ExecArgvIsExeced(t *testing.T) {
	t.Parallel()

	const selectedKey = "github.com/kalbasit/swm"

	wantArgv := []string{"/usr/bin/tmux", "-S", "/tmp/feat-x.sock", "attach-session", "-t", "swm"}

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{
		Name: testStoryName,
		Projects: []coreStory.Project{
			{Host: testHost, Segments: []string{testOwner, testSegment}},
		},
	}}

	var gotArgv []string

	testExec := workspace.ExecFunc(func(_ string, argv []string, _ []string) error {
		gotArgv = argv

		return nil
	})

	sess := &stubSess{switchToExecArgv: wantArgv}
	picker := &stubPickerClient{selectedKey: selectedKey}
	mgr := &stubMgr{sess: sess, picker: picker}
	resolver := layout.NewResolver(testCodeRoot)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop, workspace.WithExecFunc(testExec))
	cmd.SetArgs([]string{testStoryFlag, testStoryName})

	require.NoError(t, cmd.Execute())
	require.Equal(t, wantArgv, gotArgv, "expected execFunc to be called with the argv from SwitchTo")
}

func TestOpenCmd_NoPicker_FallsBackToPhase1(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{
		Name: testStoryName,
		Projects: []coreStory.Project{
			{Host: testHost, Segments: []string{testOwner, testSegment}},
		},
	}}
	sess := &stubSess{}
	mgr := &stubMgr{sess: sess} // no picker configured
	resolver := layout.NewResolver(testCodeRoot)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute())

	// Phase 1 path: OpenWorkspace with all attached projects.
	require.NotNil(t, sess.lastOpenReq)
	require.Contains(t, sess.lastOpenReq.GetWorktreePaths(), "github.com/kalbasit/swm")
}

// ─── stubs ───────────────────────────────────────────────────────────────────

// stubStore is a minimal story.Store.
type stubStore struct {
	getStory     *coreStory.Story
	getErr       error
	updateCalled bool
}

func (s *stubStore) Create(context.Context, string, string) (*coreStory.Story, error) {
	panic("stub")
}

func (s *stubStore) Delete(context.Context, string) error { return nil }

func (s *stubStore) Get(_ context.Context, _ string) (*coreStory.Story, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}

	if s.getStory != nil {
		return s.getStory, nil
	}

	return &coreStory.Story{}, nil
}

func (s *stubStore) List(context.Context) ([]*coreStory.Story, error) { return nil, nil }

func (s *stubStore) Update(_ context.Context, _ *coreStory.Story) error {
	s.updateCalled = true

	return nil
}

var _ coreStory.Store = (*stubStore)(nil)

// stubMgr implements pluginManager.
type stubMgr struct {
	sess   pluginv1.SessionClient
	vcs    pluginv1.VCSClient
	picker pluginv1.PickerClient
}

func (s *stubMgr) Get(_ context.Context, capability string) (any, error) {
	switch capability {
	case "session":
		if s.sess != nil {
			return s.sess, nil
		}
	case "vcs":
		if s.vcs != nil {
			return s.vcs, nil
		}
	case "picker":
		if s.picker != nil {
			return s.picker, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", errNoPlugin, capability)
}

// stubSess records session plugin calls.
type stubSess struct {
	lastOpenReq      *pluginv1.OpenWorkspaceRequest
	lastPaneGroupReq *pluginv1.OpenPaneGroupRequest
	switchToExecArgv []string // returned from SwitchTo when non-nil
}

func (s *stubSess) CloseWorkspace(
	context.Context,
	*pluginv1.CloseWorkspaceRequest,
	...grpc.CallOption,
) (*pluginv1.Empty, error) {
	panic("stub")
}

func (s *stubSess) CurrentContext(
	context.Context,
	*pluginv1.Empty,
	...grpc.CallOption,
) (*pluginv1.CurrentContextResponse, error) {
	panic("stub")
}

func (s *stubSess) Info(
	context.Context,
	*pluginv1.Empty,
	...grpc.CallOption,
) (*pluginv1.SessionInfo, error) {
	panic("stub")
}

func (s *stubSess) IsInsideWorkspace(
	context.Context,
	*pluginv1.Empty,
	...grpc.CallOption,
) (*pluginv1.BoolValue, error) {
	panic("stub")
}

func (s *stubSess) ListWorkspaces(
	context.Context,
	*pluginv1.Empty,
	...grpc.CallOption,
) (grpc.ServerStreamingClient[pluginv1.Workspace], error) {
	return &eofStream{}, nil
}

func (s *stubSess) OpenPaneGroup(
	_ context.Context,
	req *pluginv1.OpenPaneGroupRequest,
	_ ...grpc.CallOption,
) (*pluginv1.PaneGroup, error) {
	s.lastPaneGroupReq = req

	return &pluginv1.PaneGroup{
		PaneGroupId: "swm",
		WorkspaceId: req.GetWorkspaceId(),
	}, nil
}

func (s *stubSess) OpenWorkspace(
	_ context.Context,
	req *pluginv1.OpenWorkspaceRequest,
	_ ...grpc.CallOption,
) (*pluginv1.Workspace, error) {
	s.lastOpenReq = req

	return &pluginv1.Workspace{
		WorkspaceId: "/tmp/feat-x.sock",
		StoryName:   req.GetStoryName(),
	}, nil
}

func (s *stubSess) SwitchTo(
	context.Context,
	*pluginv1.SwitchToRequest,
	...grpc.CallOption,
) (*pluginv1.SwitchToResponse, error) {
	return &pluginv1.SwitchToResponse{ExecArgv: s.switchToExecArgv}, nil
}

var _ pluginv1.SessionClient = (*stubSess)(nil)

// stubVCS records CreateWorktree calls.
type stubVCS struct {
	createCalled bool
}

func (v *stubVCS) Clone(context.Context, *pluginv1.CloneRequest, ...grpc.CallOption) (*pluginv1.CloneResponse, error) {
	panic("stub")
}

func (v *stubVCS) CreateWorktree(
	_ context.Context,
	_ *pluginv1.CreateWorktreeRequest,
	_ ...grpc.CallOption,
) (*pluginv1.Empty, error) {
	v.createCalled = true

	return &pluginv1.Empty{}, nil
}

func (v *stubVCS) DetectProjectAtPath(
	context.Context,
	*pluginv1.DetectAtPathRequest,
	...grpc.CallOption,
) (*pluginv1.ProjectID, error) {
	panic("stub")
}

func (v *stubVCS) Info(context.Context, *pluginv1.Empty, ...grpc.CallOption) (*pluginv1.VCSInfo, error) {
	panic("stub")
}

func (v *stubVCS) ListBranches(
	context.Context,
	*pluginv1.ListBranchesRequest,
	...grpc.CallOption,
) (grpc.ServerStreamingClient[pluginv1.Branch], error) {
	panic("stub")
}

func (v *stubVCS) ParseRemoteURL(
	context.Context,
	*pluginv1.ParseRemoteURLRequest,
	...grpc.CallOption,
) (*pluginv1.ProjectID, error) {
	panic("stub")
}

func (v *stubVCS) RemoveWorktree(
	context.Context,
	*pluginv1.RemoveWorktreeRequest,
	...grpc.CallOption,
) (*pluginv1.Empty, error) {
	panic("stub")
}

var _ pluginv1.VCSClient = (*stubVCS)(nil)

// stubPickerClient implements pluginv1.PickerClient.
type stubPickerClient struct {
	selectedKey  string
	cancelOnRecv bool
	pickErr      error
	recvErr      error // returned from stream.Recv() instead of a normal result
}

func (p *stubPickerClient) Info(
	context.Context,
	*pluginv1.Empty,
	...grpc.CallOption,
) (*pluginv1.PickerInfo, error) {
	return &pluginv1.PickerInfo{PluginInfo: &pluginv1.PluginInfo{Name: "stub"}}, nil
}

func (p *stubPickerClient) Pick(
	_ context.Context,
	_ ...grpc.CallOption,
) (grpc.BidiStreamingClient[pluginv1.PickItem, pluginv1.PickResult], error) {
	if p.pickErr != nil {
		return nil, p.pickErr
	}

	return &stubPickStream{selectedKey: p.selectedKey, cancel: p.cancelOnRecv, recvErr: p.recvErr}, nil
}

var _ pluginv1.PickerClient = (*stubPickerClient)(nil)

// stubPickStream implements grpc.BidiStreamingClient[PickItem, PickResult].
type stubPickStream struct {
	selectedKey string
	cancel      bool
	recvCalled  bool
	recvErr     error // returned from Recv() when set, before checking cancel/selectedKey
}

func (s *stubPickStream) CloseSend() error { return nil }

func (s *stubPickStream) Context() context.Context { return context.Background() }

func (s *stubPickStream) Header() (metadata.MD, error) { panic("stub") }

func (s *stubPickStream) Recv() (*pluginv1.PickResult, error) {
	if s.recvCalled {
		return nil, io.EOF
	}

	s.recvCalled = true

	if s.recvErr != nil {
		return nil, s.recvErr
	}

	if s.cancel {
		return nil, status.Error(codes.Aborted, "cancelled")
	}

	return &pluginv1.PickResult{Key: s.selectedKey}, nil
}

func (s *stubPickStream) RecvMsg(any) error { return nil }

func (s *stubPickStream) Send(*pluginv1.PickItem) error { return nil }

func (s *stubPickStream) SendMsg(any) error { return nil }

func (s *stubPickStream) Trailer() metadata.MD { return nil }

// eofStream is a server stream that immediately returns io.EOF.
type eofStream struct{}

func (e *eofStream) CloseSend() error                   { return nil }
func (e *eofStream) Context() context.Context           { return context.Background() }
func (e *eofStream) Header() (metadata.MD, error)       { panic("stub") }
func (e *eofStream) Recv() (*pluginv1.Workspace, error) { return nil, io.EOF }
func (e *eofStream) RecvMsg(any) error                  { panic("stub") }
func (e *eofStream) SendMsg(any) error                  { panic("stub") }
func (e *eofStream) Trailer() metadata.MD               { panic("stub") }
