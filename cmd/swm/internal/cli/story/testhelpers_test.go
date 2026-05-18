package story_test

import (
	"context"
	"errors"
	"io"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"
	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"
)

const (
	testGitHubHost  = "github.com"
	testKalbasitOrg = "kalbasit"
	testSWMRepo     = "swm"
	testStoryName   = "feat-x"
	testBugName     = "my-bug"
	testForceFlag   = "--force"
)

// errNotFound is a sentinel used in tests.
var errNotFound = errors.New("not found")

// stubStore is a minimal story.Store implementation for CLI tests.
type stubStore struct {
	lastCreatedName   string
	lastCreatedBranch string
	createErr         error
	getStory          *coreStory.Story
	getErr            error
	deleteErr         error
	deleted           bool
	listStories       []*coreStory.Story
	listErr           error
}

func (s *stubStore) Create(_ context.Context, name, branch string) (*coreStory.Story, error) {
	s.lastCreatedName = name
	s.lastCreatedBranch = branch

	if s.createErr != nil {
		return nil, s.createErr
	}

	return &coreStory.Story{Name: name, BranchName: branch}, nil
}

func (s *stubStore) Delete(_ context.Context, _ string) error {
	s.deleted = true

	return s.deleteErr
}

func (s *stubStore) Get(_ context.Context, _ string) (*coreStory.Story, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}

	if s.getStory != nil {
		return s.getStory, nil
	}

	return &coreStory.Story{}, nil
}

func (s *stubStore) List(_ context.Context) ([]*coreStory.Story, error) {
	return s.listStories, s.listErr
}

func (s *stubStore) Update(_ context.Context, _ *coreStory.Story) error { return nil }

var _ coreStory.Store = (*stubStore)(nil)

// stubManager implements the pluginManager interface for tests.
type stubManager struct {
	vcs  pluginv1.VCSClient
	sess pluginv1.SessionClient
}

func (s *stubManager) Get(_ context.Context, capability string) (any, error) {
	switch capability {
	case "vcs":
		if s.vcs != nil {
			return s.vcs, nil
		}
	case "session":
		if s.sess != nil {
			return s.sess, nil
		}
	}

	return nil, errNotFound
}

// stubVCSClient implements pluginv1.VCSClient for tests.
type stubVCSClient struct {
	removeWorktreeCalled bool
	parseRemoteURLFn     func(*pluginv1.ParseRemoteURLRequest) (*pluginv1.ProjectID, error)
	cloneFn              func(*pluginv1.CloneRequest) (*pluginv1.CloneResponse, error)
}

func (s *stubVCSClient) Clone(
	_ context.Context,
	req *pluginv1.CloneRequest,
	_ ...grpc.CallOption,
) (*pluginv1.CloneResponse, error) {
	if s.cloneFn != nil {
		return s.cloneFn(req)
	}

	return &pluginv1.CloneResponse{}, nil
}

func (s *stubVCSClient) CreateWorktree(
	context.Context,
	*pluginv1.CreateWorktreeRequest,
	...grpc.CallOption,
) (*pluginv1.Empty, error) {
	panic("stub")
}

func (s *stubVCSClient) DetectProjectAtPath(
	context.Context,
	*pluginv1.DetectAtPathRequest,
	...grpc.CallOption,
) (*pluginv1.ProjectID, error) {
	panic("stub")
}

func (s *stubVCSClient) Info(
	context.Context,
	*pluginv1.Empty,
	...grpc.CallOption,
) (*pluginv1.VCSInfo, error) {
	panic("stub")
}

func (s *stubVCSClient) ListBranches(
	context.Context,
	*pluginv1.ListBranchesRequest,
	...grpc.CallOption,
) (grpc.ServerStreamingClient[pluginv1.Branch], error) {
	panic("stub")
}

func (s *stubVCSClient) ParseRemoteURL(
	_ context.Context,
	req *pluginv1.ParseRemoteURLRequest,
	_ ...grpc.CallOption,
) (*pluginv1.ProjectID, error) {
	if s.parseRemoteURLFn != nil {
		return s.parseRemoteURLFn(req)
	}

	return &pluginv1.ProjectID{Host: testGitHubHost, Segments: []string{testKalbasitOrg, testSWMRepo}}, nil
}

func (s *stubVCSClient) RemoveWorktree(
	_ context.Context,
	_ *pluginv1.RemoveWorktreeRequest,
	_ ...grpc.CallOption,
) (*pluginv1.Empty, error) {
	s.removeWorktreeCalled = true

	return &pluginv1.Empty{}, nil
}

var _ pluginv1.VCSClient = (*stubVCSClient)(nil)

// stubSessionClient implements pluginv1.SessionClient for tests.
type stubSessionClient struct {
	openWorkspaceFn func(*pluginv1.OpenWorkspaceRequest) (*pluginv1.Workspace, error)
}

func (s *stubSessionClient) CloseWorkspace(
	context.Context,
	*pluginv1.CloseWorkspaceRequest,
	...grpc.CallOption,
) (*pluginv1.Empty, error) {
	return &pluginv1.Empty{}, nil
}

func (s *stubSessionClient) CurrentContext(
	context.Context,
	*pluginv1.Empty,
	...grpc.CallOption,
) (*pluginv1.CurrentContextResponse, error) {
	panic("stub")
}

func (s *stubSessionClient) Info(
	context.Context,
	*pluginv1.Empty,
	...grpc.CallOption,
) (*pluginv1.SessionInfo, error) {
	panic("stub")
}

func (s *stubSessionClient) IsInsideWorkspace(
	context.Context,
	*pluginv1.Empty,
	...grpc.CallOption,
) (*pluginv1.BoolValue, error) {
	panic("stub")
}

func (s *stubSessionClient) ListWorkspaces(
	context.Context,
	*pluginv1.Empty,
	...grpc.CallOption,
) (grpc.ServerStreamingClient[pluginv1.Workspace], error) {
	return &emptyWorkspaceStream{}, nil
}

func (s *stubSessionClient) OpenPaneGroup(
	context.Context,
	*pluginv1.OpenPaneGroupRequest,
	...grpc.CallOption,
) (*pluginv1.PaneGroup, error) {
	panic("stub")
}

func (s *stubSessionClient) OpenWorkspace(
	_ context.Context,
	req *pluginv1.OpenWorkspaceRequest,
	_ ...grpc.CallOption,
) (*pluginv1.Workspace, error) {
	if s.openWorkspaceFn != nil {
		return s.openWorkspaceFn(req)
	}

	return &pluginv1.Workspace{WorkspaceId: "sock-" + req.GetStoryName(), StoryName: req.GetStoryName()}, nil
}

func (s *stubSessionClient) SwitchTo(
	context.Context,
	*pluginv1.SwitchToRequest,
	...grpc.CallOption,
) (*pluginv1.SwitchToResponse, error) {
	panic("stub")
}

var _ pluginv1.SessionClient = (*stubSessionClient)(nil)

// emptyWorkspaceStream is a grpc.ServerStreamingClient[Workspace] that returns EOF immediately.
type emptyWorkspaceStream struct{}

func (e *emptyWorkspaceStream) CloseSend() error                   { return nil }
func (e *emptyWorkspaceStream) Context() context.Context           { return context.Background() }
func (e *emptyWorkspaceStream) Header() (metadata.MD, error)       { panic("stub") }
func (e *emptyWorkspaceStream) Recv() (*pluginv1.Workspace, error) { return nil, io.EOF }
func (e *emptyWorkspaceStream) RecvMsg(any) error                  { panic("stub") }
func (e *emptyWorkspaceStream) SendMsg(any) error                  { panic("stub") }
func (e *emptyWorkspaceStream) Trailer() metadata.MD               { panic("stub") }
