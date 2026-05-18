package session_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/plugins/session-tmux/internal/session"
)

const (
	testHost          = "github.com"
	testOrg           = "kalbasit"
	testRepo          = "swm"
	testProject       = testHost + "/" + testOrg + "/" + testRepo
	testWorktree      = "/tmp/wt"
	testPaneGroup     = testRepo
	testPaneGroupFull = "github•com/kalbasit/swm"

	// testLaioPaneGroupCommandTOML is the canonical pane_group_command used in tests.
	testLaioPaneGroupCommandTOML = `pane_group_command = "laio start` +
		` --file {{worktree_path}}/.swm/laio.yaml` +
		` --socket {{tmux_socket}} --skip-attach"`
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
	t.Setenv("FAKETMUX_SESSION", testRepo)

	ctx, err := tmux.CurrentContext(context.Background(), &pluginv1.Empty{})
	require.NoError(t, err)
	require.Equal(t, sock, ctx.GetWorkspaceId())
	require.Equal(t, "mywork", ctx.GetStoryName())
	require.Equal(t, testRepo, ctx.GetPaneGroupId())
}

func TestCurrentContext_NotInside(t *testing.T) {
	// Cannot be parallel — sets env vars.
	t.Setenv("TMUX", "")

	tmux, _ := newTmux(t)
	_, err := tmux.CurrentContext(context.Background(), &pluginv1.Empty{})
	require.Error(t, err)
}

func TestSwitchTo_OutsideTmux_ReturnsExecArgv(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)
	t.Setenv("TMUX", "")

	tmux, socketDir := newTmux(t)
	sock := filepath.Join(socketDir, "feat-x.sock")

	// Create the socket file so the workspace is considered open.
	require.NoError(t, os.WriteFile(sock, nil, 0o600))

	resp, err := tmux.SwitchTo(context.Background(), &pluginv1.SwitchToRequest{
		WorkspaceId: sock,
		PaneGroupId: testPaneGroup,
	})
	require.NoError(t, err)
	require.Equal(t, []string{faketmuxBin, "-S", sock, "attach-session", "-t", testPaneGroup}, resp.GetExecArgv())

	// faketmux must NOT have been invoked — log file absent or empty.
	logBytes, readErr := os.ReadFile(logFile) //nolint:gosec // G304: test-controlled path
	if readErr == nil {
		require.Empty(t, strings.TrimSpace(string(logBytes)), "faketmux must not be called when returning exec_argv")
	} else {
		require.True(t, os.IsNotExist(readErr), "unexpected read error: %v", readErr)
	}
}

func TestSwitchTo_InsideTmux_CallsSwitchClient(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)
	sock := filepath.Join(socketDir, "feat-x.sock")
	t.Setenv("TMUX", sock+",12345,0")

	// Create the socket file so the workspace is considered open.
	require.NoError(t, os.WriteFile(sock, nil, 0o600))

	resp, err := tmux.SwitchTo(context.Background(), &pluginv1.SwitchToRequest{
		WorkspaceId: sock,
		PaneGroupId: testPaneGroup,
	})
	require.NoError(t, err)
	require.Empty(t, resp.GetExecArgv(), "exec_argv must be empty when switch-client is used")

	logBytes, err := os.ReadFile(logFile) //nolint:gosec // G304: test-controlled path
	require.NoError(t, err)
	require.Contains(t, string(logBytes), "switch-client", "faketmux must be called with switch-client")
}

func TestOpenWorkspace_SetsSWMStory(t *testing.T) {
	// Cannot be parallel — uses FAKETMUX_LOG env var.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, _ := newTmux(t)
	_, err := tmux.OpenWorkspace(context.Background(), &pluginv1.OpenWorkspaceRequest{
		StoryName:     "my-feature",
		WorktreePaths: map[string]string{},
	})
	require.NoError(t, err)

	logBytes, err := os.ReadFile(logFile) //nolint:gosec // G304: test-controlled path
	require.NoError(t, err)

	log := string(logBytes)

	// set-environment -g propagates SWM_STORY to all pane-group sessions created afterward.
	require.Contains(t, log, "set-environment -g SWM_STORY my-feature",
		"tmux set-environment must be called to propagate SWM_STORY into the workspace")

	// The bootstrap new-session must carry -e so the very first shell sees it
	// before set-environment -g has been called.
	require.Contains(t, log, "new-session",
		"expected a new-session invocation for the bootstrap session")
	require.Contains(t, log, "-e SWM_STORY=my-feature",
		"bootstrap new-session must pass SWM_STORY via -e so the first shell sees it immediately")
}

func TestOpenWorkspace_EmptyWorktreePaths(t *testing.T) {
	// Cannot be parallel — uses FAKETMUX_LOG env var.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, _ := newTmux(t)

	ws, err := tmux.OpenWorkspace(context.Background(), &pluginv1.OpenWorkspaceRequest{
		StoryName:     "story-only",
		WorktreePaths: map[string]string{},
	})
	require.NoError(t, err)
	require.Equal(t, "story-only", ws.GetStoryName())

	logBytes, err := os.ReadFile(logFile) //nolint:gosec // G304: test-controlled path
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(logBytes)), "\n")

	var firstNewSession string

	for _, line := range lines {
		if strings.Contains(line, "new-session") {
			firstNewSession = line

			break
		}
	}

	require.NotEmpty(t, firstNewSession, "expected a bootstrap new-session invocation in log")
	require.Contains(t, firstNewSession, "-s story-only",
		"bootstrap session name must be the story name when no worktree paths are provided")
	require.NotContains(t, firstNewSession, "-c ",
		"no -c flag should be present for the bootstrap session")
}

func TestOpenWorkspace_NoProjectSessionsPreCreated(t *testing.T) {
	// Cannot be parallel — uses FAKETMUX_LOG env var.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, _ := newTmux(t)

	_, err := tmux.OpenWorkspace(context.Background(), &pluginv1.OpenWorkspaceRequest{
		StoryName: "feat-order",
		WorktreePaths: map[string]string{
			"github.com/z-repo": "/tmp/stories/feat-order/z-repo",
			"github.com/a-repo": "/tmp/stories/feat-order/a-repo",
			"github.com/m-repo": "/tmp/stories/feat-order/m-repo",
		},
	})
	require.NoError(t, err)

	logBytes, err := os.ReadFile(logFile) //nolint:gosec // G304: test-controlled path
	require.NoError(t, err)

	log := string(logBytes)

	// Only the bootstrap session (story name) must appear — never a project key.
	// Sessions for project worktrees are created by OpenPaneGroup so that
	// pane_group_command is applied to each one individually.
	require.Contains(t, log, "-s feat-order",
		"bootstrap session must use the story name")
	require.NotContains(t, log, "github•com",
		"OpenWorkspace must not pre-create sessions for project worktree paths")
}

func TestOpenPaneGroup_WithPaneGroupCommand(t *testing.T) {
	// Cannot be parallel — uses t.Setenv.
	socketDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	sockPath := filepath.Join(socketDir, "feat-x.sock")

	client := &fakeHostClient{
		toml: []byte(testLaioPaneGroupCommandTOML),
	}

	tmux := session.NewWithBinAndClient(faketmuxBin, socketDir, client)

	// First create the workspace socket so has-session has a socket to probe.
	if err := os.WriteFile(sockPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := tmux.OpenPaneGroup(context.Background(), &pluginv1.OpenPaneGroupRequest{
		WorkspaceId:  sockPath,
		ProjectId:    &pluginv1.ProjectID{Host: testHost, Segments: []string{testOrg, testRepo}},
		WorktreePath: "/tmp/stories/feat-x/github.com/kalbasit/swm",
	})
	require.NoError(t, err)

	// Read the log to verify the substituted command was passed to faketmux.
	logBytes, err := os.ReadFile(logFile) //nolint:gosec // G304: test-controlled path
	require.NoError(t, err)

	wantCmd := "laio start --file /tmp/stories/feat-x/github.com/kalbasit/swm/.swm/laio.yaml" +
		" --socket " + sockPath + " --skip-attach"

	log := string(logBytes)
	require.Contains(t, log, wantCmd, "expected substituted pane_group_command in tmux args")
}

func TestOpenPaneGroup_WithPaneGroupCommand_SocketSubstitution(t *testing.T) {
	// Cannot be parallel — uses t.Setenv.
	socketDir := t.TempDir()
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	sockPath := filepath.Join(socketDir, "feat-sock.sock")

	client := &fakeHostClient{
		toml: []byte(testLaioPaneGroupCommandTOML),
	}

	tmux := session.NewWithBinAndClient(faketmuxBin, socketDir, client)

	if err := os.WriteFile(sockPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := tmux.OpenPaneGroup(context.Background(), &pluginv1.OpenPaneGroupRequest{
		WorkspaceId:  sockPath,
		ProjectId:    &pluginv1.ProjectID{Host: testHost, Segments: []string{testOrg, testRepo}},
		WorktreePath: "/tmp/stories/feat-sock/github.com/kalbasit/swm",
	})
	require.NoError(t, err)

	logBytes, err := os.ReadFile(logFile) //nolint:gosec // G304: test-controlled path
	require.NoError(t, err)

	log := string(logBytes)
	require.Contains(t, log, "--socket "+sockPath,
		"{{tmux_socket}} must be substituted with the workspace socket path")
}

func TestOpenPaneGroup_InvalidProjectID(t *testing.T) {
	t.Parallel()

	tmux, socketDir := newTmux(t)
	sockPath := filepath.Join(socketDir, "feat-x.sock")

	cases := []struct {
		name string
		pid  *pluginv1.ProjectID
	}{
		{
			name: "empty host",
			pid:  &pluginv1.ProjectID{Host: "", Segments: []string{testOrg, testRepo}},
		},
		{
			name: "empty segments",
			pid:  &pluginv1.ProjectID{Host: testHost, Segments: []string{}},
		},
		{
			name: "nil project_id",
			pid:  nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := tmux.OpenPaneGroup(context.Background(), &pluginv1.OpenPaneGroupRequest{
				WorkspaceId:  sockPath,
				ProjectId:    tc.pid,
				WorktreePath: testWorktree,
			})
			require.Error(t, err, "expected error for incomplete project_id")
		})
	}
}

func TestOpenPaneGroup_SessionNameIsFullPath(t *testing.T) {
	// Cannot be parallel — uses FAKETMUX_LOG env var.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)
	sockPath := filepath.Join(socketDir, "feat-x.sock")

	pg, err := tmux.OpenPaneGroup(context.Background(), &pluginv1.OpenPaneGroupRequest{
		WorkspaceId:  sockPath,
		ProjectId:    &pluginv1.ProjectID{Host: testHost, Segments: []string{testOrg, testRepo}},
		WorktreePath: testWorktree,
	})
	require.NoError(t, err)
	require.Equal(t, testPaneGroupFull, pg.GetPaneGroupId(),
		"pane_group_id must be the sanitized full canonical path, not just the basename")

	logBytes, err := os.ReadFile(logFile) //nolint:gosec // G304: test-controlled path
	require.NoError(t, err)
	require.Contains(t, string(logBytes), "-s "+testPaneGroupFull,
		"tmux new-session must use the full sanitized path as the session name")
}

func TestOpenPaneGroup_CollisionPrevention(t *testing.T) {
	// Cannot be parallel — uses FAKETMUX_LOG env var.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)
	sockPath := filepath.Join(socketDir, "feat-x.sock")

	pgA, err := tmux.OpenPaneGroup(context.Background(), &pluginv1.OpenPaneGroupRequest{
		WorkspaceId:  sockPath,
		ProjectId:    &pluginv1.ProjectID{Host: testHost, Segments: []string{"org-a", "utils"}},
		WorktreePath: testWorktree,
	})
	require.NoError(t, err)

	pgB, err := tmux.OpenPaneGroup(context.Background(), &pluginv1.OpenPaneGroupRequest{
		WorkspaceId:  sockPath,
		ProjectId:    &pluginv1.ProjectID{Host: testHost, Segments: []string{"org-b", "utils"}},
		WorktreePath: testWorktree,
	})
	require.NoError(t, err)

	require.NotEqual(t, pgA.GetPaneGroupId(), pgB.GetPaneGroupId(),
		"repos with the same basename from different orgs must have distinct session names")
	require.NotEqual(t, "utils", pgA.GetPaneGroupId(), "session name must not be a bare basename")
	require.NotEqual(t, "utils", pgB.GetPaneGroupId(), "session name must not be a bare basename")
}

func TestOpenWorkspace_EnvIsolation_PluginInternalVarsAbsent(t *testing.T) {
	// Cannot be parallel — sets env vars.
	envFile := filepath.Join(t.TempDir(), "env.log")
	t.Setenv("FAKETMUX_ENV_LOG", envFile)

	t.Setenv("SWM_HOST_SOCKET", "unix:///run/user/1000/swm/test.sock")
	t.Setenv("SWM_LOG_LEVEL", "debug")
	t.Setenv("SWM_PLUGIN_MAGIC_COOKIE", "swm-plugin-v1")

	tmux, _ := newTmux(t)
	_, err := tmux.OpenWorkspace(context.Background(), &pluginv1.OpenWorkspaceRequest{
		StoryName:     "env-isolation-test",
		WorktreePaths: map[string]string{},
	})
	require.NoError(t, err)

	envBytes, err := os.ReadFile(envFile) //nolint:gosec // G304: test-controlled path
	require.NoError(t, err)

	envContents := string(envBytes)
	require.NotContains(t, envContents, "SWM_HOST_SOCKET=")
	require.NotContains(t, envContents, "SWM_LOG_LEVEL=")
	require.NotContains(t, envContents, "SWM_PLUGIN_MAGIC_COOKIE=")
}

func TestOpenWorkspace_EnvIsolation_UserEnvPreserved(t *testing.T) {
	// Cannot be parallel — sets env vars.
	envFile := filepath.Join(t.TempDir(), "env.log")
	t.Setenv("FAKETMUX_ENV_LOG", envFile)

	const (
		sentinelKey = "SWM_TEST_USER_SENTINEL"
		sentinelVal = "user-env-must-survive-12345"
	)
	t.Setenv(sentinelKey, sentinelVal)

	tmux, _ := newTmux(t)
	_, err := tmux.OpenWorkspace(context.Background(), &pluginv1.OpenWorkspaceRequest{
		StoryName:     "user-env-test",
		WorktreePaths: map[string]string{},
	})
	require.NoError(t, err)

	envBytes, err := os.ReadFile(envFile) //nolint:gosec // G304: test-controlled path
	require.NoError(t, err)

	envContents := string(envBytes)
	require.Contains(t, envContents, sentinelKey+"="+sentinelVal)
	require.Contains(t, envContents, "HOME=")
	require.Contains(t, envContents, "PATH=")
}

func TestSwitchTo_InsideTmux_KillsOriginPane(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)
	originSock := filepath.Join(socketDir, "origin.sock")
	targetSock := filepath.Join(socketDir, "feat-x.sock")
	t.Setenv("TMUX", targetSock+",12345,0")

	require.NoError(t, os.WriteFile(originSock, nil, 0o600))
	require.NoError(t, os.WriteFile(targetSock, nil, 0o600))

	resp, err := tmux.SwitchTo(context.Background(), &pluginv1.SwitchToRequest{
		WorkspaceId:            targetSock,
		PaneGroupId:            testPaneGroup,
		CloseOriginWorkspaceId: originSock,
		CloseOriginPaneId:      "%5",
	})
	require.NoError(t, err)
	require.Empty(t, resp.GetExecArgv())

	logBytes, err := os.ReadFile(logFile) //nolint:gosec // G304: test-controlled path
	require.NoError(t, err)

	log := string(logBytes)
	require.Contains(t, log, "switch-client", "must call switch-client")
	require.Contains(t, log, "kill-pane", "must call kill-pane on origin")
	require.Contains(t, log, "%5", "must target the origin pane")
}

func TestSwitchTo_OutsideTmux_KillsOriginPane(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)
	t.Setenv("TMUX", "")

	tmux, socketDir := newTmux(t)
	originSock := filepath.Join(socketDir, "origin.sock")
	targetSock := filepath.Join(socketDir, "feat-x.sock")

	require.NoError(t, os.WriteFile(originSock, nil, 0o600))
	require.NoError(t, os.WriteFile(targetSock, nil, 0o600))

	resp, err := tmux.SwitchTo(context.Background(), &pluginv1.SwitchToRequest{
		WorkspaceId:            targetSock,
		PaneGroupId:            testPaneGroup,
		CloseOriginWorkspaceId: originSock,
		CloseOriginPaneId:      "%5",
	})
	require.NoError(t, err)
	require.Equal(t, []string{faketmuxBin, "-S", targetSock, "attach-session", "-t", testPaneGroup}, resp.GetExecArgv())

	logBytes, err := os.ReadFile(logFile) //nolint:gosec // G304: test-controlled path
	require.NoError(t, err)

	log := string(logBytes)
	require.Contains(t, log, "kill-pane", "must call kill-pane on origin before returning exec_argv")
	require.Contains(t, log, "%5", "must target the origin pane")
}

func TestSwitchTo_KillOriginPane_PaneAlreadyGone(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)
	t.Setenv("FAKETMUX_KILL_PANE_FAIL", "1")

	tmux, socketDir := newTmux(t)
	originSock := filepath.Join(socketDir, "origin.sock")
	targetSock := filepath.Join(socketDir, "feat-x.sock")
	t.Setenv("TMUX", targetSock+",12345,0")

	require.NoError(t, os.WriteFile(originSock, nil, 0o600))
	require.NoError(t, os.WriteFile(targetSock, nil, 0o600))

	// kill-pane fails (pane already gone) — SwitchTo must still succeed.
	_, err := tmux.SwitchTo(context.Background(), &pluginv1.SwitchToRequest{
		WorkspaceId:            targetSock,
		PaneGroupId:            testPaneGroup,
		CloseOriginWorkspaceId: originSock,
		CloseOriginPaneId:      "%5",
	})
	require.NoError(t, err, "kill-pane not-found error must be swallowed")
}

func TestSwitchTo_KillOriginPane_UnknownWorkspace(t *testing.T) {
	// Cannot be parallel — sets env vars.
	t.Setenv("TMUX", "")

	tmux, socketDir := newTmux(t)
	targetSock := filepath.Join(socketDir, "feat-x.sock")
	require.NoError(t, os.WriteFile(targetSock, nil, 0o600))

	_, err := tmux.SwitchTo(context.Background(), &pluginv1.SwitchToRequest{
		WorkspaceId:            targetSock,
		PaneGroupId:            testPaneGroup,
		CloseOriginWorkspaceId: "/nonexistent/origin.sock",
		CloseOriginPaneId:      "%5",
	})
	require.Error(t, err, "unknown origin workspace must return an error")
	require.Contains(t, err.Error(), "NotFound")
}

func TestSwitchTo_NoKill_WhenOriginPaneIdEmpty(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)
	sock := filepath.Join(socketDir, "feat-x.sock")
	t.Setenv("TMUX", sock+",12345,0")

	require.NoError(t, os.WriteFile(sock, nil, 0o600))

	_, err := tmux.SwitchTo(context.Background(), &pluginv1.SwitchToRequest{
		WorkspaceId: sock,
		PaneGroupId: testPaneGroup,
		// CloseOriginPaneId intentionally empty
	})
	require.NoError(t, err)

	logBytes, err := os.ReadFile(logFile) //nolint:gosec // G304: test-controlled path
	require.NoError(t, err)
	require.NotContains(t, string(logBytes), "kill-pane", "must not call kill-pane when origin pane id is empty")
}

// fakeHostClient implements pluginv1.HostClient for tests.
type fakeHostClient struct {
	toml []byte
}

func (c *fakeHostClient) CallCapability(
	_ context.Context,
	_ *pluginv1.CallCapabilityRequest,
	_ ...grpc.CallOption,
) (*pluginv1.CallCapabilityResponse, error) {
	panic("stub")
}

func (c *fakeHostClient) GetCodeRoot(
	_ context.Context,
	_ *pluginv1.Empty,
	_ ...grpc.CallOption,
) (*pluginv1.PathResponse, error) {
	panic("stub")
}

func (c *fakeHostClient) GetConfig(
	_ context.Context,
	_ *pluginv1.GetConfigRequest,
	_ ...grpc.CallOption,
) (*pluginv1.Config, error) {
	return &pluginv1.Config{Toml: c.toml}, nil
}

func (c *fakeHostClient) GetCurrentStory(
	_ context.Context,
	_ *pluginv1.Empty,
	_ ...grpc.CallOption,
) (*pluginv1.Story, error) {
	panic("stub")
}

func (c *fakeHostClient) ListProjects(
	_ context.Context,
	_ *pluginv1.ListProjectsRequest,
	_ ...grpc.CallOption,
) (grpc.ServerStreamingClient[pluginv1.Project], error) {
	panic("stub")
}

func (c *fakeHostClient) Log(
	_ context.Context,
	_ *pluginv1.LogRequest,
	_ ...grpc.CallOption,
) (*pluginv1.Empty, error) {
	panic("stub")
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
