package workspace_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"
	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/cmd/swm/internal/cli/workspace"
	"github.com/kalbasit/swm/cmd/swm/internal/config"
	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
)

const (
	testCodeRoot     = "/code"
	testDefaultStory = "_default"
	testStoryName    = "feat-x"
	testStoryFlag    = "--story"
)

var errNoPlugin = errors.New("no plugin")

func TestOpenCmd_WithStoryFlag(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{Name: testStoryName}}
	sess := &stubSess{}
	mgr := &stubMgr{sess: sess}
	resolver := layout.NewResolver(testCodeRoot)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver)
	cmd.SetArgs([]string{testStoryFlag, testStoryName})

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

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver)
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

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver)
	cmd.SetArgs([]string{testStoryFlag, "nonexistent"})

	require.Error(t, cmd.Execute())
}

func TestOpenCmd_NoProjects(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{Name: testStoryName, Projects: nil}}
	sess := &stubSess{}
	mgr := &stubMgr{sess: sess}
	resolver := layout.NewResolver(testCodeRoot)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver)
	cmd.SetArgs([]string{testStoryFlag, testStoryName})

	require.NoError(t, cmd.Execute())
	require.Empty(t, sess.lastOpenReq.GetWorktreePaths())
}

// stubStore is a minimal story.Store.
type stubStore struct {
	getStory *coreStory.Story
	getErr   error
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

func (s *stubStore) Update(context.Context, *coreStory.Story) error { return nil }

var _ coreStory.Store = (*stubStore)(nil)

// stubMgr implements pluginManager.
type stubMgr struct {
	sess pluginv1.SessionClient
}

func (s *stubMgr) Get(_ context.Context, capability string) (any, error) {
	if capability == "session" && s.sess != nil {
		return s.sess, nil
	}

	return nil, fmt.Errorf("%w: %s", errNoPlugin, capability)
}

// stubSess records OpenWorkspace calls.
type stubSess struct {
	lastOpenReq *pluginv1.OpenWorkspaceRequest
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
	context.Context,
	*pluginv1.OpenPaneGroupRequest,
	...grpc.CallOption,
) (*pluginv1.PaneGroup, error) {
	panic("stub")
}

func (s *stubSess) OpenWorkspace(
	_ context.Context,
	req *pluginv1.OpenWorkspaceRequest,
	_ ...grpc.CallOption,
) (*pluginv1.Workspace, error) {
	s.lastOpenReq = req

	return &pluginv1.Workspace{StoryName: req.GetStoryName()}, nil
}

func (s *stubSess) SwitchTo(
	context.Context,
	*pluginv1.SwitchToRequest,
	...grpc.CallOption,
) (*pluginv1.Empty, error) {
	panic("stub")
}

var _ pluginv1.SessionClient = (*stubSess)(nil)

type eofStream struct{}

func (e *eofStream) CloseSend() error                   { return nil }
func (e *eofStream) Context() context.Context           { return context.Background() }
func (e *eofStream) Header() (metadata.MD, error)       { panic("stub") }
func (e *eofStream) Recv() (*pluginv1.Workspace, error) { return nil, io.EOF }
func (e *eofStream) RecvMsg(any) error                  { panic("stub") }
func (e *eofStream) SendMsg(any) error                  { panic("stub") }
func (e *eofStream) Trailer() metadata.MD               { panic("stub") }
