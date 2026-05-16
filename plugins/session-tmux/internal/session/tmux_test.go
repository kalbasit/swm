package session_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/plugins/session-tmux/internal/session"
)

const (
	testProject  = "github.com/kalbasit/swm"
	testWorktree = "/tmp/wt"
)

var faketmuxBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "session-tmux-faketmux-*")
	if err != nil {
		panic("create temp dir: " + err.Error())
	}

	defer os.RemoveAll(dir) //nolint:errcheck // best-effort cleanup in TestMain

	faketmuxBin = filepath.Join(dir, "faketmux")

	buildCmd := exec.Command("go", "build", "-o", faketmuxBin, "./testdata/faketmux") //nolint:gosec // test build

	out, err := buildCmd.CombinedOutput()
	if err != nil {
		panic("build faketmux: " + string(out))
	}

	os.Exit(m.Run())
}

func newTmux(t *testing.T) (*session.Tmux, string) {
	t.Helper()
	socketDir := t.TempDir()

	return session.NewWithBin(faketmuxBin, socketDir), socketDir
}

func TestInfo(t *testing.T) {
	t.Parallel()

	tmux, _ := newTmux(t)
	info, err := tmux.Info(context.Background(), &pluginv1.Empty{})
	require.NoError(t, err)
	require.Equal(t, "tmux", info.GetPluginInfo().GetName())
}

func TestOpenWorkspace(t *testing.T) {
	t.Parallel()

	tmux, _ := newTmux(t)
	ws, err := tmux.OpenWorkspace(context.Background(), &pluginv1.OpenWorkspaceRequest{
		StoryName: "feat-x",
		WorktreePaths: map[string]string{
			testProject: "/tmp/stories/feat-x/" + testProject,
		},
	})
	require.NoError(t, err)
	require.Equal(t, "feat-x", ws.GetStoryName())
	require.NotEmpty(t, ws.GetWorkspaceId())
}

func TestOpenWorkspace_Idempotent(t *testing.T) {
	t.Parallel()

	tmux, _ := newTmux(t)
	req := &pluginv1.OpenWorkspaceRequest{
		StoryName:     "feat-y",
		WorktreePaths: map[string]string{testProject: testWorktree},
	}

	// First open creates the workspace.
	ws1, err := tmux.OpenWorkspace(context.Background(), req)
	require.NoError(t, err)

	// Second open attaches to the same workspace (socket already exists).
	ws2, err := tmux.OpenWorkspace(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, ws1.GetWorkspaceId(), ws2.GetWorkspaceId())
}

func TestCloseWorkspace(t *testing.T) {
	t.Parallel()

	tmux, socketDir := newTmux(t)
	ws, err := tmux.OpenWorkspace(context.Background(), &pluginv1.OpenWorkspaceRequest{
		StoryName:     "close-me",
		WorktreePaths: map[string]string{testProject: testWorktree},
	})
	require.NoError(t, err)

	_, err = tmux.CloseWorkspace(context.Background(), &pluginv1.CloseWorkspaceRequest{
		WorkspaceId: ws.GetWorkspaceId(),
	})
	require.NoError(t, err)

	// Socket file should be gone.
	_, err = os.Stat(filepath.Join(socketDir, "close-me.sock"))
	require.True(t, os.IsNotExist(err))
}

func TestCloseWorkspace_Idempotent(t *testing.T) {
	t.Parallel()

	tmux, _ := newTmux(t)
	// Close a workspace that was never opened — should not error.
	_, err := tmux.CloseWorkspace(context.Background(), &pluginv1.CloseWorkspaceRequest{
		WorkspaceId: "/nonexistent/path.sock",
	})
	require.NoError(t, err)
}

func TestListWorkspaces(t *testing.T) {
	t.Parallel()

	tmux, _ := newTmux(t)

	// Open two workspaces.
	for _, story := range []string{"alpha", "beta"} {
		_, err := tmux.OpenWorkspace(context.Background(), &pluginv1.OpenWorkspaceRequest{
			StoryName:     story,
			WorktreePaths: map[string]string{testProject: testWorktree},
		})
		require.NoError(t, err)
	}

	stream := &collectWorkspaceStream{ctx: context.Background()}
	err := tmux.ListWorkspaces(&pluginv1.Empty{}, stream)
	require.NoError(t, err)
	require.Len(t, stream.items, 2)
}

func TestIsInsideWorkspace_Outside(t *testing.T) {
	// Cannot be parallel — sets env vars.
	t.Setenv("TMUX", "")

	tmux, _ := newTmux(t)
	result, err := tmux.IsInsideWorkspace(context.Background(), &pluginv1.Empty{})
	require.NoError(t, err)
	require.False(t, result.GetValue())
}

func TestIsInsideWorkspace_Inside(t *testing.T) {
	// Cannot be parallel — sets env vars.
	tmux, socketDir := newTmux(t)
	sock := filepath.Join(socketDir, "feat-z.sock")
	t.Setenv("TMUX", sock+",12345,0")

	result, err := tmux.IsInsideWorkspace(context.Background(), &pluginv1.Empty{})
	require.NoError(t, err)
	require.True(t, result.GetValue())
}

func TestCurrentContext(t *testing.T) {
	// Cannot be parallel — sets env vars.
	tmux, socketDir := newTmux(t)
	sock := filepath.Join(socketDir, "mywork.sock")
	t.Setenv("TMUX", sock+",12345,0")
	t.Setenv("FAKETMUX_SESSION", "swm")

	ctx, err := tmux.CurrentContext(context.Background(), &pluginv1.Empty{})
	require.NoError(t, err)
	require.Equal(t, sock, ctx.GetWorkspaceId())
	require.Equal(t, "mywork", ctx.GetStoryName())
	require.Equal(t, "swm", ctx.GetPaneGroupId())
}

func TestCurrentContext_NotInside(t *testing.T) {
	// Cannot be parallel — sets env vars.
	t.Setenv("TMUX", "")

	tmux, _ := newTmux(t)
	_, err := tmux.CurrentContext(context.Background(), &pluginv1.Empty{})
	require.Error(t, err)
}

// collectWorkspaceStream implements pluginv1.Session_ListWorkspacesServer for tests.
type collectWorkspaceStream struct {
	pluginv1.Session_ListWorkspacesServer
	ctx   context.Context
	items []*pluginv1.Workspace
}

func (s *collectWorkspaceStream) Context() context.Context { return s.ctx }

func (s *collectWorkspaceStream) Send(ws *pluginv1.Workspace) error {
	s.items = append(s.items, ws)

	return nil
}
