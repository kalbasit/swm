package session_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/plugins/session-tmux/internal/session"
)

const (
	newWindowCmd = "new-window"
	sendKeysCmd  = "send-keys"
	killPaneCmd  = "kill-pane"

	noWorkspace = "no workspace"
	helloText   = "hello"

	testPaneID    = "%1"
	otherPaneID   = "%2"
	missingPaneID = "%99"
)

// fakePaneRow renders one faketmux list-panes row in the field order the plugin
// asks tmux for: pane ID, session, title, current command, current path,
// session_attached, window_active, pane_active.
func fakePaneRow(id, group, title, command, path, attached, winActive, paneActive string) string {
	return strings.Join([]string{id, group, title, command, path, attached, winActive, paneActive}, "\t")
}

// unfocusedPaneRow is a pane on a workspace no client is attached to, so no
// human's keystrokes can be reaching it.
func unfocusedPaneRow(id, group string) string {
	return fakePaneRow(id, group, "zsh", "zsh", testWorktree, "0", "1", "1")
}

// focusedPaneRow is a pane an attached client is currently typing into.
func focusedPaneRow(id, group string) string {
	return fakePaneRow(id, group, "zsh", "zsh", testWorktree, "1", "1", "1")
}

// seedWorkspace creates a live socket carrying the given pane rows.
func seedWorkspace(t *testing.T, socketDir, story string, rows ...string) string {
	t.Helper()

	sock := filepath.Join(socketDir, story+".sock")
	require.NoError(t, os.WriteFile(sock, nil, 0o600))
	require.NoError(t, os.WriteFile(sock+".panes", []byte(strings.Join(rows, "\n")+"\n"), 0o600))

	return sock
}

// seedDeadWorkspace creates a socket file whose server has exited. faketmux
// treats a socket containing "dead" as one no server is listening on.
func seedDeadWorkspace(t *testing.T, socketDir, story string) string {
	t.Helper()

	sock := filepath.Join(socketDir, story+".sock")
	require.NoError(t, os.WriteFile(sock, []byte("dead"), 0o600))

	return sock
}

// tmuxLogLines returns the faketmux invocations recorded so far. A log that was
// never created means no invocation happened at all.
func tmuxLogLines(t *testing.T, logFile string) []string {
	t.Helper()

	b, err := os.ReadFile(logFile) //nolint:gosec // G304: test-controlled path
	if os.IsNotExist(err) {
		return nil
	}

	require.NoError(t, err)

	return strings.Split(strings.TrimSpace(string(b)), "\n")
}

// tmuxLogLine returns the single recorded invocation containing want.
func tmuxLogLine(t *testing.T, logFile, want string) string {
	t.Helper()

	var found []string

	for _, l := range tmuxLogLines(t, logFile) {
		if strings.Contains(l, want) {
			found = append(found, l)
		}
	}

	require.Len(t, found, 1, "expected exactly one %q invocation in %v", want, tmuxLogLines(t, logFile))

	return found[0]
}

type collectPaneStream struct {
	pluginv1.Session_ListPanesServer

	ctx   context.Context
	items []*pluginv1.Pane
}

func (s *collectPaneStream) Context() context.Context { return s.ctx }

func (s *collectPaneStream) Send(p *pluginv1.Pane) error {
	s.items = append(s.items, p)

	return nil
}

func listPanes(t *testing.T, tmux *session.Tmux, req *pluginv1.ListPanesRequest) []*pluginv1.Pane {
	t.Helper()

	stream := &collectPaneStream{ctx: context.Background()}
	require.NoError(t, tmux.ListPanes(req, stream))

	return stream.items
}

func TestListPanes_AllWorkspaces(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)
	seedWorkspace(t, socketDir, "alpha", unfocusedPaneRow(testPaneID, testPaneGroupFull))
	seedWorkspace(t, socketDir, "beta", unfocusedPaneRow(testPaneID, testPaneGroupFull))

	panes := listPanes(t, tmux, &pluginv1.ListPanesRequest{})
	require.Len(t, panes, 2)

	// Each pane must carry the workspace it came from, not the one asked for —
	// an empty request asked for none.
	require.Equal(t, filepath.Join(socketDir, "alpha.sock"), panes[0].GetWorkspaceId())
	require.Equal(t, filepath.Join(socketDir, "beta.sock"), panes[1].GetWorkspaceId())
}

func TestListPanes_FilterByWorkspace(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)
	sockA := seedWorkspace(t, socketDir, "alpha", unfocusedPaneRow(testPaneID, testPaneGroupFull))
	seedWorkspace(t, socketDir, "beta", unfocusedPaneRow(testPaneID, testPaneGroupFull))

	panes := listPanes(t, tmux, &pluginv1.ListPanesRequest{WorkspaceId: sockA})
	require.Len(t, panes, 1)
	require.Equal(t, sockA, panes[0].GetWorkspaceId())
}

func TestListPanes_FilterByPaneGroup(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)
	sock := seedWorkspace(t, socketDir, "alpha",
		unfocusedPaneRow(testPaneID, testPaneGroupFull),
		unfocusedPaneRow(otherPaneID, prefixNameTwo),
	)

	panes := listPanes(t, tmux, &pluginv1.ListPanesRequest{WorkspaceId: sock, PaneGroupId: prefixNameTwo})
	require.Len(t, panes, 1)
	require.Equal(t, otherPaneID, panes[0].GetPaneId())
	require.Equal(t, prefixNameTwo, panes[0].GetPaneGroupId())
}

func TestListPanes_DescriptiveFields(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)
	sock := seedWorkspace(t, socketDir, "alpha",
		fakePaneRow(testPaneID, testPaneGroupFull, "a title", "some-tool", "/srv/wt", "0", "1", "1"),
	)

	panes := listPanes(t, tmux, &pluginv1.ListPanesRequest{WorkspaceId: sock})
	require.Len(t, panes, 1)

	p := panes[0]
	require.Equal(t, testPaneID, p.GetPaneId())
	require.Equal(t, testPaneGroupFull, p.GetPaneGroupId())
	require.Equal(t, "a title", p.GetTitle())
	require.Equal(t, "some-tool", p.GetCurrentCommand())
	require.Equal(t, "/srv/wt", p.GetCurrentPath())
}

func TestListPanes_FocusReported(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)
	sock := seedWorkspace(t, socketDir, "alpha",
		focusedPaneRow(testPaneID, testPaneGroupFull),
		// Attached, but not the active pane: keystrokes land elsewhere.
		fakePaneRow(otherPaneID, testPaneGroupFull, "zsh", "zsh", testWorktree, "1", "1", "0"),
		// Active pane of a window the attached client is not showing.
		fakePaneRow("%3", testPaneGroupFull, "zsh", "zsh", testWorktree, "1", "0", "1"),
	)

	panes := listPanes(t, tmux, &pluginv1.ListPanesRequest{WorkspaceId: sock})
	require.Len(t, panes, 3)
	require.True(t, panes[0].GetFocused())
	require.False(t, panes[1].GetFocused(), "another pane in the same window has the keystrokes")
	require.False(t, panes[2].GetFocused(), "the client is not showing this window")
}

func TestListPanes_UnattachedIsNotFocused(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)
	sock := seedWorkspace(t, socketDir, "alpha", unfocusedPaneRow(testPaneID, testPaneGroupFull))

	panes := listPanes(t, tmux, &pluginv1.ListPanesRequest{WorkspaceId: sock})
	require.Len(t, panes, 1)
	require.False(t, panes[0].GetFocused(), "no client is attached, so nobody can be typing here")
}

func TestListPanes_UnknownWorkspace(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)

	stream := &collectPaneStream{ctx: context.Background()}
	err := tmux.ListPanes(&pluginv1.ListPanesRequest{
		WorkspaceId: filepath.Join(socketDir, "nope.sock"),
	}, stream)

	require.Equal(t, codes.NotFound, status.Code(err))
	require.Empty(t, stream.items)
}

func TestListPanes_DeadSocketSkipped(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)
	seedDeadWorkspace(t, socketDir, "stale")
	seedWorkspace(t, socketDir, "zlive", unfocusedPaneRow(testPaneID, testPaneGroupFull))

	panes := listPanes(t, tmux, &pluginv1.ListPanesRequest{})
	require.Len(t, panes, 1)
	require.Equal(t, filepath.Join(socketDir, "zlive.sock"), panes[0].GetWorkspaceId())
}

func TestOpenPane_QuotesArgv(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)
	sock := filepath.Join(socketDir, "feat-x.sock")

	pane, err := tmux.OpenPane(context.Background(), &pluginv1.OpenPaneRequest{
		WorkspaceId: sock,
		PaneGroupId: testPaneGroupFull,
		Argv:        []string{"my-tool", "--flag", "two words"},
		Cwd:         testWorktree,
	})
	require.NoError(t, err)
	require.Equal(t, testPaneID, pane.GetPaneId(), "pane_id must be the handle tmux minted")
	require.Equal(t, testPaneGroupFull, pane.GetPaneGroupId())
	require.Equal(t, sock, pane.GetWorkspaceId())

	require.Equal(t,
		"-S "+sock+" "+newWindowCmd+" -P -F #{pane_id} -t ="+testPaneGroupFull+
			" -c "+testWorktree+" my-tool --flag 'two words'",
		tmuxLogLine(t, logFile, newWindowCmd),
	)
}

func TestOpenPane_EmptyArgvStartsShell(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)
	sock := filepath.Join(socketDir, "feat-x.sock")

	_, err := tmux.OpenPane(context.Background(), &pluginv1.OpenPaneRequest{
		WorkspaceId: sock,
		PaneGroupId: testPaneGroupFull,
	})
	require.NoError(t, err)

	// No trailing command at all: tmux then runs the default shell.
	require.Equal(t,
		"-S "+sock+" "+newWindowCmd+" -P -F #{pane_id} -t ="+testPaneGroupFull,
		tmuxLogLine(t, logFile, newWindowCmd),
	)
}

func TestOpenPane_EnvIsDeterministic(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)
	sock := filepath.Join(socketDir, "feat-x.sock")

	_, err := tmux.OpenPane(context.Background(), &pluginv1.OpenPaneRequest{
		WorkspaceId: sock,
		PaneGroupId: testPaneGroupFull,
		Env:         map[string]string{"ZED": "3", "ALPHA": "1", "MIKE": "2"},
	})
	require.NoError(t, err)

	// Go randomises map iteration; the emitted command line must not inherit that.
	require.Contains(t, tmuxLogLine(t, logFile, newWindowCmd), "-e ALPHA=1 -e MIKE=2 -e ZED=3")
}

func TestOpenPane_TargetsPaneGroupExactly(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)
	sock := filepath.Join(socketDir, "feat-x.sock")

	_, err := tmux.OpenPane(context.Background(), &pluginv1.OpenPaneRequest{
		WorkspaceId: sock,
		PaneGroupId: prefixName,
	})
	require.NoError(t, err)

	// "=" prefix stops tmux resolving the name to a longer one it prefixes.
	require.Contains(t, tmuxLogLine(t, logFile, newWindowCmd), "-t ="+prefixName)
}

func TestOpenPane_UnknownPaneGroup(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)
	t.Setenv("FAKETMUX_NEW_WINDOW_FAIL", "1")

	tmux, socketDir := newTmux(t)

	_, err := tmux.OpenPane(context.Background(), &pluginv1.OpenPaneRequest{
		WorkspaceId: filepath.Join(socketDir, "feat-x.sock"),
		PaneGroupId: testPaneGroupFull,
	})
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestOpenPane_MissingIdentifiers(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)
	sock := filepath.Join(socketDir, "feat-x.sock")

	for name, req := range map[string]*pluginv1.OpenPaneRequest{
		noWorkspace:     {PaneGroupId: testPaneGroupFull},
		"no pane group": {WorkspaceId: sock},
	} {
		_, err := tmux.OpenPane(context.Background(), req)
		require.Equal(t, codes.InvalidArgument, status.Code(err), name)
	}

	require.Empty(t, tmuxLogLines(t, logFile), "a rejected request must not reach tmux")
}

func TestSendText_TextThenSubmit(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)
	sock := seedWorkspace(t, socketDir, "feat-x", unfocusedPaneRow(testPaneID, testPaneGroupFull))

	_, err := tmux.SendText(context.Background(), &pluginv1.SendTextRequest{
		WorkspaceId: sock,
		PaneId:      testPaneID,
		Text:        helloText,
		Submit:      true,
	})
	require.NoError(t, err)

	var sends []string

	for _, l := range tmuxLogLines(t, logFile) {
		if strings.Contains(l, sendKeysCmd) {
			sends = append(sends, l)
		}
	}

	require.Equal(t, []string{
		"-S " + sock + " " + sendKeysCmd + " -t " + testPaneID + " -l -- " + helloText,
		"-S " + sock + " " + sendKeysCmd + " -t " + testPaneID + " Enter",
	}, sends, "text is delivered literally, and the submit key is a separate delivery")
}

func TestSendText_TextThatLooksLikeAKeyName(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)
	sock := seedWorkspace(t, socketDir, "feat-x", unfocusedPaneRow(testPaneID, testPaneGroupFull))

	_, err := tmux.SendText(context.Background(), &pluginv1.SendTextRequest{
		WorkspaceId: sock,
		PaneId:      testPaneID,
		Text:        "Enter",
	})
	require.NoError(t, err)

	// Without -l tmux would read "Enter" as the Enter key rather than as text.
	require.Equal(t,
		"-S "+sock+" "+sendKeysCmd+" -t "+testPaneID+" -l -- Enter",
		tmuxLogLine(t, logFile, sendKeysCmd),
	)
}

func TestSendText_TextWithLeadingDash(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)
	sock := seedWorkspace(t, socketDir, "feat-x", unfocusedPaneRow(testPaneID, testPaneGroupFull))

	_, err := tmux.SendText(context.Background(), &pluginv1.SendTextRequest{
		WorkspaceId: sock,
		PaneId:      testPaneID,
		Text:        "-n oops",
	})
	require.NoError(t, err)

	// "--" ends option parsing; real tmux rejects `send-keys -l '-n oops'`.
	require.Contains(t, tmuxLogLine(t, logFile, sendKeysCmd), " -l -- -n oops")
}

func TestSendText_SubmitOnly(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)
	sock := seedWorkspace(t, socketDir, "feat-x", unfocusedPaneRow(testPaneID, testPaneGroupFull))

	_, err := tmux.SendText(context.Background(), &pluginv1.SendTextRequest{
		WorkspaceId: sock,
		PaneId:      testPaneID,
		Submit:      true,
	})
	require.NoError(t, err)

	require.Equal(t,
		"-S "+sock+" "+sendKeysCmd+" -t "+testPaneID+" Enter",
		tmuxLogLine(t, logFile, sendKeysCmd),
		"empty text with submit delivers only the submit key",
	)
}

func TestSendText_NothingToSend(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)
	sock := seedWorkspace(t, socketDir, "feat-x", unfocusedPaneRow(testPaneID, testPaneGroupFull))

	before := len(tmuxLogLines(t, logFile))

	_, err := tmux.SendText(context.Background(), &pluginv1.SendTextRequest{
		WorkspaceId: sock,
		PaneId:      testPaneID,
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Len(t, tmuxLogLines(t, logFile), before, "a rejected request must not reach tmux")
}

func TestSendText_MissingIdentifiers(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)
	sock := filepath.Join(socketDir, "feat-x.sock")

	for name, req := range map[string]*pluginv1.SendTextRequest{
		noWorkspace: {PaneId: testPaneID, Text: "hi"},
		"no pane":   {WorkspaceId: sock, Text: "hi"},
	} {
		_, err := tmux.SendText(context.Background(), req)
		require.Equal(t, codes.InvalidArgument, status.Code(err), name)
	}
}

func TestSendText_UnknownPane(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)
	sock := seedWorkspace(t, socketDir, "feat-x", unfocusedPaneRow(testPaneID, testPaneGroupFull))

	_, err := tmux.SendText(context.Background(), &pluginv1.SendTextRequest{
		WorkspaceId: sock,
		PaneId:      missingPaneID,
		Text:        helloText,
	})
	require.Equal(t, codes.NotFound, status.Code(err))

	for _, l := range tmuxLogLines(t, logFile) {
		require.NotContains(t, l, sendKeysCmd, "nothing may be delivered to a pane that does not exist")
	}
}

func TestSendText_FocusedPaneRefused(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)
	sock := seedWorkspace(t, socketDir, "feat-x", focusedPaneRow(testPaneID, testPaneGroupFull))

	_, err := tmux.SendText(context.Background(), &pluginv1.SendTextRequest{
		WorkspaceId: sock,
		PaneId:      testPaneID,
		Text:        "y",
		Submit:      true,
	})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
	require.Contains(t, status.Convert(err).Message(), "allow_focused",
		"the refusal must name the override so the caller knows the way out")

	for _, l := range tmuxLogLines(t, logFile) {
		require.NotContains(t, l, sendKeysCmd, "a refused send must deliver nothing")
	}
}

func TestSendText_FocusedPaneWithOverride(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)
	sock := seedWorkspace(t, socketDir, "feat-x", focusedPaneRow(testPaneID, testPaneGroupFull))

	_, err := tmux.SendText(context.Background(), &pluginv1.SendTextRequest{
		WorkspaceId:  sock,
		PaneId:       testPaneID,
		Text:         helloText,
		AllowFocused: true,
	})
	require.NoError(t, err)
	require.Contains(t, tmuxLogLine(t, logFile, sendKeysCmd), "-l -- "+helloText)
}

func TestSendText_DelayObservedBeforeDelivery(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	const delayMs = 120

	tmux, socketDir := newTmux(t)
	sock := seedWorkspace(t, socketDir, "feat-x", unfocusedPaneRow(testPaneID, testPaneGroupFull))

	start := time.Now()

	_, err := tmux.SendText(context.Background(), &pluginv1.SendTextRequest{
		WorkspaceId: sock,
		PaneId:      testPaneID,
		Text:        helloText,
		DelayMs:     delayMs,
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, time.Since(start), delayMs*time.Millisecond)
}

func TestSendText_CancelledDuringDelay(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)
	sock := seedWorkspace(t, socketDir, "feat-x", unfocusedPaneRow(testPaneID, testPaneGroupFull))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := tmux.SendText(ctx, &pluginv1.SendTextRequest{
		WorkspaceId: sock,
		PaneId:      testPaneID,
		Text:        helloText,
		DelayMs:     60_000,
	})
	require.ErrorIs(t, err, context.Canceled)

	for _, l := range tmuxLogLines(t, logFile) {
		require.NotContains(t, l, sendKeysCmd, "a cancelled wait must deliver nothing")
	}
}

func TestClosePane(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)
	sock := filepath.Join(socketDir, "feat-x.sock")

	_, err := tmux.ClosePane(context.Background(), &pluginv1.ClosePaneRequest{
		WorkspaceId: sock,
		PaneId:      testPaneID,
	})
	require.NoError(t, err)

	// A pane ID is already unambiguous, so it is never escaped for exact matching.
	require.Equal(t,
		"-S "+sock+" "+killPaneCmd+" -t "+testPaneID,
		tmuxLogLine(t, logFile, killPaneCmd),
	)
}

func TestClosePane_AlreadyGone(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)
	t.Setenv("FAKETMUX_KILL_PANE_FAIL", "1")

	tmux, socketDir := newTmux(t)

	_, err := tmux.ClosePane(context.Background(), &pluginv1.ClosePaneRequest{
		WorkspaceId: filepath.Join(socketDir, "feat-x.sock"),
		PaneId:      testPaneID,
	})
	require.NoError(t, err, "cleaning up after a program that already exited is the normal case")
}

func TestClosePane_MissingIdentifiers(t *testing.T) {
	// Cannot be parallel — sets env vars.
	logFile := filepath.Join(t.TempDir(), "tmux.log")
	t.Setenv("FAKETMUX_LOG", logFile)

	tmux, socketDir := newTmux(t)
	sock := filepath.Join(socketDir, "feat-x.sock")

	for name, req := range map[string]*pluginv1.ClosePaneRequest{
		noWorkspace: {PaneId: testPaneID},
		"no pane":   {WorkspaceId: sock},
	} {
		_, err := tmux.ClosePane(context.Background(), req)
		require.Equal(t, codes.InvalidArgument, status.Code(err), name)
	}

	require.Empty(t, tmuxLogLines(t, logFile), "a rejected request must not reach tmux")
}
