package pane_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/cmd/swm/internal/cli/pane"
)

const (
	capSession = "session"

	cmdOpen  = "open"
	cmdList  = "list"
	cmdSend  = "send"
	cmdClose = "close"

	testWorkspaceID = "/run/user/1000/swm/tmux/feat-x.sock"
	testPaneGroupID = "github.com/kalbasit/swm"
	testPaneID      = "%4"
	missingPaneID   = "%999"
	testWorktree    = "/home/user/code/stories/feat-x/github.com/kalbasit/swm"
	testCommand     = "nvim"
	testText        = "hello"

	flagWorkspace = "--workspace"
	flagPaneGroup = "--pane-group"
	flagPane      = "--pane"
	flagJSON      = "--json"
	flagSubmit    = "--submit"

	caseNoFlags       = "no flags"
	caseWorkspaceOnly = "workspace only"
	casePaneOnly      = "pane only"
	caseNoSession     = "no session plugin"
)

// errNoSessionPlugin stands in for the manager failing to resolve the session
// capability.
var errNoSessionPlugin = errors.New("no session plugin configured")

// errPluginFailure is a generic plugin-side failure with no gRPC status.
var errPluginFailure = errors.New("plugin exploded")

// errFocused is the refusal a Session plugin returns for a focused pane.
var errFocused = status.Error(codes.FailedPrecondition,
	"pane %4 is focused by an attached client")

// errPaneNotFound is the status a plugin returns for an unknown pane.
var errPaneNotFound = status.Error(codes.NotFound, "pane not found")

// runPane executes the `swm pane` group with args, returning stdout and the
// command error.
func runPane(t *testing.T, mgr pane.PluginManager, args ...string) (string, error) {
	t.Helper()

	var out bytes.Buffer

	cmd := pane.NewPaneCmd(mgr)
	cmd.SetArgs(args)
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	// Execute first: evaluating out.String() in the return statement would read
	// the buffer before the command has written to it.
	err := cmd.Execute()

	return out.String(), err
}

// wantPaneJSON is the exact object the pane commands must emit for a pane.
// Every key is always present, including the descriptive ones a provider may
// leave empty — that is the contract external callers index against.
func wantPaneJSON(paneID, groupID, workspaceID, title, command, path string, focused bool) map[string]any {
	return map[string]any{
		"pane_id":         paneID,
		"pane_group_id":   groupID,
		"workspace_id":    workspaceID,
		"title":           title,
		"current_command": command,
		"current_path":    path,
		"focused":         focused,
	}
}

// stubManager implements the pane package's PluginManager.
type stubManager struct {
	sess    pluginv1.SessionClient
	getErr  error
	warmErr error
}

func (s *stubManager) Get(_ context.Context, capability string) (any, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}

	if capability != capSession || s.sess == nil {
		return nil, errNoSessionPlugin
	}

	return s.sess, nil
}

func (s *stubManager) Warm(context.Context, ...string) error { return s.warmErr }

// wrongPluginManager resolves the session capability to something that is not
// a SessionClient.
type wrongPluginManager struct{}

func (w *wrongPluginManager) Get(context.Context, string) (any, error) { return "not a client", nil }

func (w *wrongPluginManager) Warm(context.Context, ...string) error { return nil }

// stubSessionClient records the pane requests it receives and replays canned
// responses.
type stubSessionClient struct {
	openReq  *pluginv1.OpenPaneRequest
	openPane *pluginv1.Pane
	openErr  error

	listReq   *pluginv1.ListPanesRequest
	listPanes []*pluginv1.Pane
	listErr   error
	recvErr   error

	sendReq *pluginv1.SendTextRequest
	sendErr error

	closeReq    *pluginv1.ClosePaneRequest
	closeErr    error
	closeCalled bool
}

func (s *stubSessionClient) ClosePane(
	_ context.Context,
	req *pluginv1.ClosePaneRequest,
	_ ...grpc.CallOption,
) (*pluginv1.Empty, error) {
	s.closeCalled = true
	s.closeReq = req

	if s.closeErr != nil {
		return nil, s.closeErr
	}

	return &pluginv1.Empty{}, nil
}

func (s *stubSessionClient) CloseWorkspace(
	context.Context,
	*pluginv1.CloseWorkspaceRequest,
	...grpc.CallOption,
) (*pluginv1.Empty, error) {
	panic("stub")
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

func (s *stubSessionClient) ListPanes(
	_ context.Context,
	req *pluginv1.ListPanesRequest,
	_ ...grpc.CallOption,
) (grpc.ServerStreamingClient[pluginv1.Pane], error) {
	s.listReq = req

	if s.listErr != nil {
		return nil, s.listErr
	}

	return &stubPaneStream{panes: s.listPanes, recvErr: s.recvErr}, nil
}

func (s *stubSessionClient) ListWorkspaces(
	context.Context,
	*pluginv1.Empty,
	...grpc.CallOption,
) (grpc.ServerStreamingClient[pluginv1.Workspace], error) {
	panic("stub")
}

func (s *stubSessionClient) OpenPane(
	_ context.Context,
	req *pluginv1.OpenPaneRequest,
	_ ...grpc.CallOption,
) (*pluginv1.Pane, error) {
	s.openReq = req

	if s.openErr != nil {
		return nil, s.openErr
	}

	if s.openPane != nil {
		return s.openPane, nil
	}

	return &pluginv1.Pane{
		PaneId:      testPaneID,
		PaneGroupId: req.GetPaneGroupId(),
		WorkspaceId: req.GetWorkspaceId(),
	}, nil
}

func (s *stubSessionClient) OpenPaneGroup(
	context.Context,
	*pluginv1.OpenPaneGroupRequest,
	...grpc.CallOption,
) (*pluginv1.PaneGroup, error) {
	panic("stub")
}

func (s *stubSessionClient) OpenWorkspace(
	context.Context,
	*pluginv1.OpenWorkspaceRequest,
	...grpc.CallOption,
) (*pluginv1.Workspace, error) {
	panic("stub")
}

func (s *stubSessionClient) SendText(
	_ context.Context,
	req *pluginv1.SendTextRequest,
	_ ...grpc.CallOption,
) (*pluginv1.Empty, error) {
	s.sendReq = req

	if s.sendErr != nil {
		return nil, s.sendErr
	}

	return &pluginv1.Empty{}, nil
}

func (s *stubSessionClient) SwitchTo(
	context.Context,
	*pluginv1.SwitchToRequest,
	...grpc.CallOption,
) (*pluginv1.SwitchToResponse, error) {
	panic("stub")
}

var _ pluginv1.SessionClient = (*stubSessionClient)(nil)

// stubPaneStream replays a fixed list of panes, optionally failing after them.
type stubPaneStream struct {
	panes   []*pluginv1.Pane
	recvErr error
	next    int
}

func (s *stubPaneStream) CloseSend() error             { return nil }
func (s *stubPaneStream) Context() context.Context     { return context.Background() }
func (s *stubPaneStream) Header() (metadata.MD, error) { panic("stub") }

func (s *stubPaneStream) Recv() (*pluginv1.Pane, error) {
	if s.next < len(s.panes) {
		p := s.panes[s.next]
		s.next++

		return p, nil
	}

	if s.recvErr != nil {
		return nil, s.recvErr
	}

	return nil, io.EOF
}

func (s *stubPaneStream) RecvMsg(any) error    { panic("stub") }
func (s *stubPaneStream) SendMsg(any) error    { panic("stub") }
func (s *stubPaneStream) Trailer() metadata.MD { panic("stub") }
