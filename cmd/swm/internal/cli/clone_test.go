package cli_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/cmd/swm/internal/cli"
	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
	"github.com/kalbasit/swm/cmd/swm/internal/hookexec"
)

const testSSHURL = "git@github.com:kalbasit/swm.git"

var (
	errNetworkError = errors.New("network error")
	errNoPlugin     = errors.New("no plugin")
)

func TestCloneCmd_Success(t *testing.T) {
	t.Parallel()

	codeRoot := t.TempDir()
	resolver := layout.NewResolver(codeRoot, "_default")
	vcs := &stubVCS{}
	mgr := &stubMgr{vcs: vcs}

	cmd := cli.NewCloneCmd(mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testSSHURL})

	require.NoError(t, cmd.Execute())
	require.True(t, vcs.cloneCalled)

	expected := filepath.Join(codeRoot, "repositories", "github.com", "kalbasit", "swm")
	require.Equal(t, expected, vcs.cloneDst)
}

func TestCloneCmd_AlreadyCloned(t *testing.T) {
	t.Parallel()

	codeRoot := t.TempDir()
	// Pre-create the canonical path with a .git directory.
	canonical := filepath.Join(codeRoot, "repositories", "github.com", "kalbasit", "swm")
	require.NoError(t, os.MkdirAll(filepath.Join(canonical, ".git"), 0o750))

	resolver := layout.NewResolver(codeRoot, "_default")
	vcs := &stubVCS{}
	mgr := &stubMgr{vcs: vcs}

	cmd := cli.NewCloneCmd(mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testSSHURL})

	require.NoError(t, cmd.Execute())
	require.False(t, vcs.cloneCalled)
}

func TestCloneCmd_CloneError(t *testing.T) {
	t.Parallel()

	codeRoot := t.TempDir()
	resolver := layout.NewResolver(codeRoot, "_default")
	vcs := &stubVCS{cloneErr: errNetworkError}
	mgr := &stubMgr{vcs: vcs}

	cmd := cli.NewCloneCmd(mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testSSHURL})

	require.Error(t, cmd.Execute())
}

// stubMgr implements cli.PluginManager.
type stubMgr struct {
	vcs  pluginv1.VCSClient
	sess pluginv1.SessionClient
}

func (s *stubMgr) Get(_ context.Context, capability string) (any, error) {
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

	return nil, fmt.Errorf("%w: %s", errNoPlugin, capability)
}

func (s *stubMgr) GetForge(_ context.Context, _ string) (pluginv1.ForgeClient, error) {
	return nil, fmt.Errorf("%w: no forge configured", errNoPlugin)
}

// stubVCS is a minimal VCSClient for clone tests.
type stubVCS struct {
	cloneCalled bool
	cloneDst    string
	cloneErr    error
}

func (s *stubVCS) Clone(
	_ context.Context,
	req *pluginv1.CloneRequest,
	_ ...grpc.CallOption,
) (*pluginv1.CloneResponse, error) {
	s.cloneCalled = true
	s.cloneDst = req.GetDestinationPath()

	if s.cloneErr != nil {
		return nil, s.cloneErr
	}

	return &pluginv1.CloneResponse{}, nil
}

func (s *stubVCS) CreateWorktree(
	context.Context,
	*pluginv1.CreateWorktreeRequest,
	...grpc.CallOption,
) (*pluginv1.Empty, error) {
	panic("stub")
}

func (s *stubVCS) DetectProjectAtPath(
	context.Context,
	*pluginv1.DetectAtPathRequest,
	...grpc.CallOption,
) (*pluginv1.ProjectID, error) {
	panic("stub")
}

func (s *stubVCS) Info(
	context.Context,
	*pluginv1.Empty,
	...grpc.CallOption,
) (*pluginv1.VCSInfo, error) {
	panic("stub")
}

func (s *stubVCS) ListBranches(
	context.Context,
	*pluginv1.ListBranchesRequest,
	...grpc.CallOption,
) (grpc.ServerStreamingClient[pluginv1.Branch], error) {
	panic("stub")
}

func (s *stubVCS) ParseRemoteURL(
	_ context.Context,
	_ *pluginv1.ParseRemoteURLRequest,
	_ ...grpc.CallOption,
) (*pluginv1.ProjectID, error) {
	return &pluginv1.ProjectID{Host: "github.com", Segments: []string{"kalbasit", "swm"}}, nil
}

func (s *stubVCS) RemoveWorktree(
	context.Context,
	*pluginv1.RemoveWorktreeRequest,
	...grpc.CallOption,
) (*pluginv1.Empty, error) {
	panic("stub")
}

var _ pluginv1.VCSClient = (*stubVCS)(nil)

// stubSessionClient implements pluginv1.SessionClient for workspace tests.
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
	return &emptyStream{}, nil
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

	return &pluginv1.Workspace{WorkspaceId: "sock", StoryName: req.GetStoryName()}, nil
}

func (s *stubSessionClient) SwitchTo(
	context.Context,
	*pluginv1.SwitchToRequest,
	...grpc.CallOption,
) (*pluginv1.SwitchToResponse, error) {
	panic("stub")
}

var _ pluginv1.SessionClient = (*stubSessionClient)(nil)

// emptyStream returns EOF immediately.
type emptyStream struct{}

func (e *emptyStream) CloseSend() error                   { return nil }
func (e *emptyStream) Context() context.Context           { return context.Background() }
func (e *emptyStream) Header() (metadata.MD, error)       { panic("stub") }
func (e *emptyStream) Recv() (*pluginv1.Workspace, error) { return nil, io.EOF }
func (e *emptyStream) RecvMsg(any) error                  { panic("stub") }
func (e *emptyStream) SendMsg(any) error                  { panic("stub") }
func (e *emptyStream) Trailer() metadata.MD               { panic("stub") }
