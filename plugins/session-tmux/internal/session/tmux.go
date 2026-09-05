// Package session implements the swm Session capability using the system tmux binary.
package session

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/adrg/xdg"
	"github.com/pelletier/go-toml/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/plugins/session-tmux/internal/layout"
	"github.com/kalbasit/swm/plugins/session-tmux/internal/shellquote"
)

// buildVersion is set via -ldflags at build time.
var buildVersion = "dev" //nolint:gochecknoglobals // set via ldflags at link time

// paneFormat is the tmux -F format behind both ListPanes and the SendText
// focus guard. Fields are tab-separated because a tab cannot occur in a pane ID,
// a session name, or a path, while a space occurs in all sorts of titles.
const paneFormat = "#{pane_id}\t#{session_name}\t#{pane_title}\t" +
	"#{pane_current_command}\t#{pane_current_path}\t" +
	"#{session_attached}\t#{window_active}\t#{pane_active}"

// paneFieldCount is how many fields paneFormat produces.
const paneFieldCount = 8

// sessionNameReplacer substitutes characters that are unsafe in tmux session names.
var sessionNameReplacer = strings.NewReplacer(".", "•", ":", "：") //nolint:gochecknoglobals // package-level replacer

// tmuxConfig holds the plugin-specific config read from the host.
type tmuxConfig struct {
	PaneGroupCommand string `toml:"pane_group_command"`
}

// Tmux implements pluginv1.SessionServer by shelling out to the system tmux.
type Tmux struct {
	tmuxBin    string
	socketDir  string
	configHome string
	hostClient pluginv1.HostClient
	grpcConn   *grpc.ClientConn
}

// New returns a Tmux instance using the system tmux binary.
// It connects to SWM_HOST_SOCKET if set, enabling host config lookups.
func New() (*Tmux, error) {
	bin, err := exec.LookPath("tmux")
	if err != nil {
		return nil, fmt.Errorf("tmux binary not found in PATH: %w", err)
	}

	t := &Tmux{
		tmuxBin:    bin,
		socketDir:  filepath.Join(xdg.RuntimeDir, "swm", "tmux"),
		configHome: xdg.ConfigHome,
	}

	if sock := os.Getenv("SWM_HOST_SOCKET"); sock != "" {
		conn, err := grpc.NewClient(
			sock,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			return nil, fmt.Errorf("connecting to host socket: %w", err)
		}

		t.grpcConn = conn
		t.hostClient = pluginv1.NewHostClient(conn)
	}

	return t, nil
}

// NewWithBin returns a Tmux instance with an injected binary path and socket dir (for tests).
func NewWithBin(tmuxBin, socketDir string) *Tmux {
	return &Tmux{tmuxBin: tmuxBin, socketDir: socketDir, configHome: xdg.ConfigHome}
}

// NewWithBinAndClient returns a Tmux instance with an injected binary, socket dir,
// and host client (for tests that exercise pane_group_command).
func NewWithBinAndClient(tmuxBin, socketDir string, client pluginv1.HostClient) *Tmux {
	return &Tmux{tmuxBin: tmuxBin, socketDir: socketDir, configHome: xdg.ConfigHome, hostClient: client}
}

// NewWithBinAndConfigHome returns a Tmux instance with an injected binary, socket dir,
// and XDG config home (for tests that exercise layout config resolution).
func NewWithBinAndConfigHome(tmuxBin, socketDir, configHome string) *Tmux {
	return &Tmux{tmuxBin: tmuxBin, socketDir: socketDir, configHome: configHome}
}

// NewWithBinClientAndConfigHome returns a Tmux instance with all dependencies injected
// (for tests that exercise both pane_group_command and layout config).
func NewWithBinClientAndConfigHome(tmuxBin, socketDir, configHome string, client pluginv1.HostClient) *Tmux {
	return &Tmux{tmuxBin: tmuxBin, socketDir: socketDir, configHome: configHome, hostClient: client}
}

// Close releases the gRPC connection to the host service.
func (t *Tmux) Close() error {
	if t.grpcConn != nil {
		return t.grpcConn.Close()
	}

	return nil
}

// ClosePane terminates a pane.
//
// A pane that is already gone is not an error: cleaning up after a program that
// exited on its own is the normal case, and a caller should not have to
// distinguish it. This matches CloseWorkspace, which is idempotent for the same
// reason.
func (t *Tmux) ClosePane(ctx context.Context, req *pluginv1.ClosePaneRequest) (*pluginv1.Empty, error) {
	sock := req.GetWorkspaceId()
	paneID := req.GetPaneId()

	if sock == "" || paneID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "workspace_id and pane_id are required")
	}

	// paneID is a tmux-assigned pane ID (%N), not a name — it is already
	// unambiguous and must not be escaped with exactTarget.
	if _, err := t.run(ctx, "-S", sock, "kill-pane", "-t", paneID); err != nil {
		if isTargetNotFound(err) {
			return &pluginv1.Empty{}, nil
		}

		return nil, err
	}

	return &pluginv1.Empty{}, nil
}

// CloseWorkspace tears down the tmux server for the given workspace.
func (t *Tmux) CloseWorkspace(ctx context.Context, req *pluginv1.CloseWorkspaceRequest) (*pluginv1.Empty, error) {
	sock := req.GetWorkspaceId()

	// Kill the tmux server; ignore errors — socket may already be gone.
	_, _ = t.run(ctx, "-S", sock, "kill-server") //nolint:errcheck // best-effort kill server
	_ = os.Remove(sock)                          //nolint:errcheck // best-effort socket cleanup

	return &pluginv1.Empty{}, nil
}

// CurrentContext returns the workspace and pane group the caller is currently inside.
func (t *Tmux) CurrentContext(ctx context.Context, _ *pluginv1.Empty) (*pluginv1.CurrentContextResponse, error) {
	tmuxEnv := os.Getenv("TMUX")
	if tmuxEnv == "" {
		return nil, status.Errorf(codes.NotFound, "not inside a tmux session")
	}

	// $TMUX is "<socket-path>,<pid>,<session-id>"
	sock, _, _ := strings.Cut(tmuxEnv, ",")
	storyName := strings.TrimSuffix(filepath.Base(sock), ".sock")

	paneGroup, err := t.run(ctx, "display-message", "-p", "#S")
	if err != nil {
		return nil, err
	}

	return &pluginv1.CurrentContextResponse{
		WorkspaceId: sock,
		StoryName:   storyName,
		PaneGroupId: paneGroup,
	}, nil
}

// Info returns metadata about this Session plugin.
func (t *Tmux) Info(_ context.Context, _ *pluginv1.Empty) (*pluginv1.SessionInfo, error) {
	return &pluginv1.SessionInfo{
		PluginInfo: &pluginv1.PluginInfo{
			Name:    "session-tmux",
			Version: buildVersion,
		},
	}, nil
}

// IsInsideWorkspace reports whether the caller is inside a swm-managed tmux workspace.
func (t *Tmux) IsInsideWorkspace(_ context.Context, _ *pluginv1.Empty) (*pluginv1.BoolValue, error) {
	tmuxEnv := os.Getenv("TMUX")
	if tmuxEnv == "" {
		return &pluginv1.BoolValue{Value: false}, nil
	}

	// $TMUX is "<socket-path>,<pid>,<session-id>"
	sock, _, _ := strings.Cut(tmuxEnv, ",")
	inside := strings.HasPrefix(sock, t.socketDir)

	return &pluginv1.BoolValue{Value: inside}, nil
}

// ListPanes streams the panes of one workspace, or of every live workspace when
// no workspace is named.
//
// This is what lets a caller ask what is actually running on this host without
// talking to the multiplexer itself, which is the whole point of swm owning
// multiplexer access.
func (t *Tmux) ListPanes(req *pluginv1.ListPanesRequest, stream pluginv1.Session_ListPanesServer) error {
	ctx := stream.Context()

	socks, err := t.workspaceSockets(ctx, req.GetWorkspaceId())
	if err != nil {
		return err
	}

	group := req.GetPaneGroupId()

	for _, sock := range socks {
		panes, err := t.panes(ctx, sock)
		if err != nil {
			return err
		}

		for _, p := range panes {
			if group != "" && p.GetPaneGroupId() != group {
				continue
			}

			if err := stream.Send(p); err != nil {
				return err
			}
		}
	}

	return nil
}

// ListWorkspaces streams all live swm tmux workspaces.
func (t *Tmux) ListWorkspaces(_ *pluginv1.Empty, stream pluginv1.Session_ListWorkspacesServer) error {
	socks, err := t.workspaceSockets(stream.Context(), "")
	if err != nil {
		return err
	}

	for _, sock := range socks {
		if err := stream.Send(&pluginv1.Workspace{
			WorkspaceId: sock,
			StoryName:   strings.TrimSuffix(filepath.Base(sock), ".sock"),
		}); err != nil {
			return err
		}
	}

	return nil
}

// OpenPane starts a program in a new pane inside a pane group.
//
// It creates a window rather than splitting an existing pane: choosing which
// pane to split and in which direction is geometry policy, which this contract
// deliberately leaves to the provider and to the layout config.
func (t *Tmux) OpenPane(ctx context.Context, req *pluginv1.OpenPaneRequest) (*pluginv1.Pane, error) {
	sock := req.GetWorkspaceId()
	group := req.GetPaneGroupId()

	if sock == "" || group == "" {
		return nil, status.Errorf(codes.InvalidArgument, "workspace_id and pane_group_id are required")
	}

	args := []string{"-S", sock, "new-window", "-P", "-F", "#{pane_id}", "-t", exactTarget(group)}

	if cwd := req.GetCwd(); cwd != "" {
		args = append(args, "-c", cwd)
	}

	// Sorted so the emitted command line is a function of the request alone.
	// Go randomises map iteration order, which would otherwise make the same
	// request produce a different tmux invocation on every call.
	env := req.GetEnv()
	for _, k := range slices.Sorted(maps.Keys(env)) {
		args = append(args, "-e", k+"="+env[k])
	}

	// tmux takes the trailing arguments as one shell command, so argv has to be
	// quoted here: without it an element containing a space would be split into
	// two arguments by the shell tmux hands it to.
	if len(req.GetArgv()) > 0 {
		args = append(args, shellquote.Argv(req.GetArgv()))
	}

	paneID, err := t.run(ctx, args...)
	if err != nil {
		if isTargetNotFound(err) {
			return nil, status.Errorf(codes.NotFound, "pane group not found on workspace %s: %s", sock, group)
		}

		return nil, err
	}

	return &pluginv1.Pane{
		PaneId:      paneID,
		PaneGroupId: group,
		WorkspaceId: sock,
	}, nil
}

// OpenPaneGroup creates or reuses a tmux session for a project inside a workspace.
func (t *Tmux) OpenPaneGroup(ctx context.Context, req *pluginv1.OpenPaneGroupRequest) (*pluginv1.PaneGroup, error) {
	sock := req.GetWorkspaceId()
	pid := req.GetProjectId()

	if pid.GetHost() == "" || len(pid.GetSegments()) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "project_id is incomplete (missing host or segments)")
	}

	name := sessionName(pid.GetHost() + "/" + strings.Join(pid.GetSegments(), "/"))

	// Determine the initial command for the session.
	initialCmd, err := t.paneGroupCommand(ctx, req)
	if err != nil {
		return nil, err
	}

	if initialCmd != "" {
		if err := validateCommandBinary(initialCmd); err != nil {
			return nil, err
		}
	}

	// Create session if it doesn't exist yet.
	if _, err := t.run(ctx, "-S", sock, "has-session", "-t", exactTarget(name)); err != nil {
		args := []string{"-S", sock, "new-session", "-d", "-s", name, "-c", req.GetWorktreePath()}
		if initialCmd != "" {
			args = append(args, initialCmd)
		}

		if _, err := t.run(ctx, args...); err != nil {
			return nil, err
		}

		if initialCmd == "" {
			if err := t.applyLayout(ctx, sock, name, req); err != nil {
				return nil, err
			}
		} else if t.layoutConfigExists(req.GetWorktreePath()) {
			log.Printf("session-tmux: pane_group_command is set; ignoring layout config for %s", name)
		}
	}

	return &pluginv1.PaneGroup{
		PaneGroupId:  name,
		WorkspaceId:  sock,
		ProjectId:    req.GetProjectId(),
		WorktreePath: req.GetWorktreePath(),
	}, nil
}

// OpenWorkspace creates or reattaches to the tmux server for the given story.
// A single bootstrap session (named after the story) is created to keep the
// server alive; project sessions are created lazily by OpenPaneGroup so that
// pane_group_command is applied to each one individually.
func (t *Tmux) OpenWorkspace(ctx context.Context, req *pluginv1.OpenWorkspaceRequest) (*pluginv1.Workspace, error) {
	sock := t.socketPath(req.GetStoryName())

	if err := os.MkdirAll(filepath.Dir(sock), 0o700); err != nil {
		return nil, status.Errorf(codes.Internal, "creating socket dir: %v", err)
	}

	// Start the server if it is not already running. A session named after the
	// story keeps the server alive (tmux exits with exit-empty=on when there are
	// no sessions). Project sessions use "host/org/repo" names and never collide
	// with the short story name used here.
	if _, err := t.run(ctx, "-S", sock, "list-sessions"); err != nil {
		bootstrapName := sessionName(req.GetStoryName())

		args := []string{"-S", sock, "new-session", "-d", "-s", bootstrapName, "-e", "SWM_STORY=" + req.GetStoryName()}
		if _, err := t.run(ctx, args...); err != nil {
			return nil, err
		}
	}

	// Propagate the story name so shells inside the workspace can run
	// "swm workspace open" without specifying --story explicitly.
	if _, err := t.run(ctx, "-S", sock, "set-environment", "-g", "SWM_STORY", req.GetStoryName()); err != nil {
		return nil, err
	}

	return &pluginv1.Workspace{
		WorkspaceId: sock,
		StoryName:   req.GetStoryName(),
	}, nil
}

// SendText delivers text to a pane, optionally followed by Enter.
//
// The delay is observed before the focus check rather than after it, so that
// the check happens as close to delivery as possible: a pane that was safe to
// write to when the call arrived may not be safe several hundred milliseconds
// later.
func (t *Tmux) SendText(ctx context.Context, req *pluginv1.SendTextRequest) (*pluginv1.Empty, error) {
	sock := req.GetWorkspaceId()
	paneID := req.GetPaneId()

	if sock == "" || paneID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "workspace_id and pane_id are required")
	}

	if req.GetText() == "" && !req.GetSubmit() {
		return nil, status.Errorf(codes.InvalidArgument, "nothing to send: text is empty and submit is not set")
	}

	if err := waitDelay(ctx, req.GetDelayMs()); err != nil {
		return nil, err
	}

	if err := t.checkPaneWritable(ctx, sock, paneID, req.GetAllowFocused()); err != nil {
		return nil, err
	}

	// -l delivers the text literally. Without it tmux reads its arguments as
	// key names, so the word "Enter" would press the Enter key and "C-c" would
	// interrupt whatever is running. "--" ends option parsing so that text
	// beginning with a dash is not read as a flag — verified against tmux 3.6a,
	// which rejects `send-keys -l '-n hello'` with "unknown flag -n".
	//
	// paneID is a tmux-assigned pane ID (%N), already unambiguous, never escaped.
	if req.GetText() != "" {
		if _, err := t.run(ctx, "-S", sock, "send-keys", "-t", paneID, "-l", "--", req.GetText()); err != nil {
			return nil, err
		}
	}

	// Submitting is a separate delivery, not a suffix on the text: Enter has to
	// arrive as a key, and the text must not.
	if req.GetSubmit() {
		if _, err := t.run(ctx, "-S", sock, "send-keys", "-t", paneID, "Enter"); err != nil {
			return nil, err
		}
	}

	return &pluginv1.Empty{}, nil
}

// SwitchTo brings the given pane group into focus.
// When the caller is already inside a tmux session, it calls switch-client directly.
// When not inside tmux, it returns exec_argv so the host can exec tmux attach-session
// with the terminal it holds — the plugin subprocess has no TTY.
//
// When close_origin_pane_id is set, the originating pane is killed inside this
// handler before the response is returned, so that the kill runs even when the
// host will subsequently syscall.Exec the returned exec_argv.
func (t *Tmux) SwitchTo(ctx context.Context, req *pluginv1.SwitchToRequest) (*pluginv1.SwitchToResponse, error) {
	sock := req.GetWorkspaceId()
	target := req.GetPaneGroupId()

	var resp *pluginv1.SwitchToResponse

	if os.Getenv("TMUX") != "" {
		if _, err := t.run(ctx, "-S", sock, "switch-client", "-t", exactTarget(target)); err != nil {
			return nil, err
		}

		resp = &pluginv1.SwitchToResponse{}
	} else {
		resp = &pluginv1.SwitchToResponse{
			ExecArgv: []string{t.tmuxBin, "-S", sock, "attach-session", "-t", exactTarget(target)},
		}
	}

	if err := t.killOriginPane(ctx, req.GetCloseOriginWorkspaceId(), req.GetCloseOriginPaneId()); err != nil {
		return nil, err
	}

	return resp, nil
}

// applyLayout resolves and applies the session-tmux layout for a newly created pane group.
// Falls back to the built-in default layout (editor + shell) when no config file exists.
func (t *Tmux) applyLayout(ctx context.Context, sock, sessionName string, req *pluginv1.OpenPaneGroupRequest) error {
	storyName := strings.TrimSuffix(filepath.Base(req.GetWorkspaceId()), ".sock")
	pid := req.GetProjectId()
	vars := layout.TemplateVars{
		WorktreePath: req.GetWorktreePath(),
		StoryName:    storyName,
		ProjectID:    pid.GetHost() + "/" + strings.Join(pid.GetSegments(), "/"),
		TmuxSocket:   req.GetWorkspaceId(),
	}

	cfg, err := layout.LoadConfig(req.GetWorktreePath(), t.configHome, vars)
	if err != nil {
		return err
	}

	if cfg == nil {
		cfg = defaultLayout()
	}

	return layout.Apply(ctx, t.run, sock, sessionName, cfg)
}

// checkPaneWritable reports whether text may be delivered into a pane.
//
// Text sent to a pane is indistinguishable from typing, so it is consumed by
// whatever the pane is currently showing — including a confirmation prompt a
// person is in the middle of answering. Refusing a pane an attached client is
// focused on turns that accident into an error the caller has to opt out of.
//
// It is a guard, not a guarantee: someone can focus the pane the instant after
// this returns. Callers must not read a successful send as proof that nobody
// was typing.
func (t *Tmux) checkPaneWritable(ctx context.Context, sock, paneID string, allowFocused bool) error {
	panes, err := t.panes(ctx, sock)
	if err != nil {
		return err
	}

	for _, p := range panes {
		if p.GetPaneId() != paneID {
			continue
		}

		if p.GetFocused() && !allowFocused {
			return status.Errorf(codes.FailedPrecondition,
				"pane %s is focused by an attached client: sending text would type into a pane someone is using; "+
					"set allow_focused to send anyway", paneID)
		}

		return nil
	}

	return status.Errorf(codes.NotFound, "pane not found on workspace %s: %s", sock, paneID)
}

// killOriginPane kills the specified pane in the origin workspace after a switch.
// It is a no-op when either argument is empty.
// "No such pane" errors from tmux are swallowed — the pane may have already closed.
func (t *Tmux) killOriginPane(ctx context.Context, originSock, paneID string) error {
	if originSock == "" || paneID == "" {
		return nil
	}

	if _, err := os.Stat(originSock); os.IsNotExist(err) {
		return status.Errorf(codes.NotFound, "origin workspace not found: %s", originSock)
	}

	// paneID is a tmux-assigned pane ID (%N), not a name — it is already
	// unambiguous and must not be escaped with exactTarget.
	if _, err := t.run(ctx, "-S", originSock, "kill-pane", "-t", paneID); err != nil {
		if isTargetNotFound(err) {
			return nil
		}

		return err
	}

	return nil
}

// isTargetNotFound reports whether a tmux error means the target it names no
// longer exists — an expected race, not a failure.
//
// tmux exits non-zero for both "the thing is gone" and "the command was wrong",
// so the message is the only signal available to tell them apart.
func isTargetNotFound(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())

	return strings.Contains(msg, "no such") ||
		strings.Contains(msg, "can't find") ||
		strings.Contains(msg, "no sessions")
}

// layoutConfigExists reports whether a layout config file exists at either tier
// (per-repo .swm/session-tmux.toml or global $XDG_CONFIG_HOME/swm/session-tmux.toml).
func (t *Tmux) layoutConfigExists(worktreePath string) bool {
	candidates := []string{
		filepath.Join(worktreePath, ".swm", "session-tmux.toml"),
		filepath.Join(t.configHome, "swm", "session-tmux.toml"),
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}

	return false
}

// paneGroupCommand returns the rendered pane_group_command string, or ("", nil) when
// no command is configured. Returns a non-nil error when the configured command's
// template is invalid or references an unknown variable.
func (t *Tmux) paneGroupCommand(ctx context.Context, req *pluginv1.OpenPaneGroupRequest) (string, error) {
	if t.hostClient == nil {
		return "", nil
	}

	resp, err := t.hostClient.GetConfig(ctx, &pluginv1.GetConfigRequest{PluginName: "session-tmux"})
	if err != nil {
		return "", nil //nolint:nilerr // host RPC failure means no config available; not a user-facing error
	}

	var cfg tmuxConfig
	if err := toml.Unmarshal(resp.GetToml(), &cfg); err != nil {
		return "", nil //nolint:nilerr // malformed TOML treated as unconfigured; host validates config
	}

	if cfg.PaneGroupCommand == "" {
		return "", nil
	}

	storyName := strings.TrimSuffix(filepath.Base(req.GetWorkspaceId()), ".sock")

	pid := req.GetProjectId()
	vars := layout.TemplateVars{
		WorktreePath: req.GetWorktreePath(),
		StoryName:    storyName,
		ProjectID:    pid.GetHost() + "/" + strings.Join(pid.GetSegments(), "/"),
		TmuxSocket:   req.GetWorkspaceId(),
	}

	tmpl, err := template.New("cmd").Option("missingkey=error").Parse(cfg.PaneGroupCommand)
	if err != nil {
		return "", status.Errorf(codes.InvalidArgument, "pane_group_command template parse error: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", status.Errorf(codes.InvalidArgument, "pane_group_command template execute error: %v", err)
	}

	return buf.String(), nil
}

// panes returns the panes tmux reports on a socket.
//
// ListPanes and the SendText focus guard both read this one query, so
// Pane.focused and the value the guard acts on cannot drift apart: a caller
// that sees focused=false from ListPanes is not then refused for a reason it
// had no way to observe.
func (t *Tmux) panes(ctx context.Context, sock string) ([]*pluginv1.Pane, error) {
	out, err := t.run(ctx, "-S", sock, "list-panes", "-a", "-F", paneFormat)
	if err != nil {
		return nil, err
	}

	if out == "" {
		return nil, nil
	}

	lines := strings.Split(out, "\n")
	panes := make([]*pluginv1.Pane, 0, len(lines))

	for _, line := range lines {
		fields := strings.Split(line, "\t")

		// A short row means the format and the parser disagree. Reporting a
		// pane whose fields are shifted would be worse than reporting none.
		if len(fields) != paneFieldCount {
			continue
		}

		panes = append(panes, &pluginv1.Pane{
			PaneId:      fields[0],
			PaneGroupId: fields[1],
			WorkspaceId: sock,
			Title:       fields[2],

			CurrentCommand: fields[3],
			CurrentPath:    fields[4],

			// A pane is where a human is typing only when a client is attached
			// to its session, that session is showing its window, and the pane
			// is the active one within that window.
			Focused: fields[5] != "0" && fields[6] == "1" && fields[7] == "1",
		})
	}

	return panes, nil
}

// validateCommandBinary checks that the first token of cmd resolves to an
// executable in PATH. Returns FailedPrecondition if not found, so callers can
// surface a clear error before handing the command to tmux.
func validateCommandBinary(cmd string) error {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return status.Errorf(codes.FailedPrecondition, "pane_group_command contains no command")
	}

	binary := fields[0]

	if _, err := exec.LookPath(binary); err != nil {
		return status.Errorf(codes.FailedPrecondition,
			"pane_group_command binary %q not found in PATH: install it or update pane_group_command in config", binary)
	}

	return nil
}

// waitDelay waits delayMs before the caller acts, aborting if the context ends
// first.
//
// This is the same mitigation the layout config exposes as pane_cmd_delay: a
// program that has only just started may not have installed its input handler
// yet, and anything typed before it does is lost. The two knobs are kept
// deliberately identical in meaning — a wait *before* the delivery, not between
// deliveries — so that a caller who has learned one has learned the other.
func waitDelay(ctx context.Context, delayMs int32) error {
	if delayMs <= 0 {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(delayMs) * time.Millisecond):
		return nil
	}
}

// defaultLayout returns the built-in two-window layout (editor + shell).
func defaultLayout() *layout.Config {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	return &layout.Config{
		Windows: []layout.Window{
			{Name: "editor", Panes: []layout.Pane{{Commands: []string{editor}}}},
			{Name: "shell"},
		},
	}
}

func (t *Tmux) run(ctx context.Context, args ...string) (string, error) {
	var stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, t.tmuxBin, args...) //nolint:gosec // tmuxBin from LookPath, args are controlled
	cmd.Env = filteredEnv()
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		return "", status.Errorf(codes.Internal, "tmux %s: %s", strings.Join(args, " "), stderr.String())
	}

	return strings.TrimSpace(string(out)), nil
}

// socketPath returns the tmux socket path for a story.
func (t *Tmux) socketPath(storyName string) string {
	return filepath.Join(t.socketDir, storyName+".sock")
}

// workspaceSockets returns the workspace sockets a request addresses.
//
// An empty workspaceID means every live workspace: the socket directory is
// scanned and each socket probed, because a socket file outlives the tmux
// server that created it and a stale one must not be reported as a workspace.
// A named workspace is returned once its socket file is known to exist; it is
// not probed, so that a failure of the caller's own command surfaces as that
// command's error rather than as a bare "not found".
func (t *Tmux) workspaceSockets(ctx context.Context, workspaceID string) ([]string, error) {
	if workspaceID != "" {
		if _, err := os.Stat(workspaceID); err != nil {
			return nil, status.Errorf(codes.NotFound, "workspace not found: %s", workspaceID)
		}

		return []string{workspaceID}, nil
	}

	entries, err := os.ReadDir(t.socketDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, status.Errorf(codes.Internal, "reading socket dir: %v", err)
	}

	socks := make([]string, 0, len(entries))

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sock") {
			continue
		}

		sock := filepath.Join(t.socketDir, e.Name())

		// Probe liveness — skip dead sockets.
		if _, err := t.run(ctx, "-S", sock, "list-sessions"); err != nil {
			continue
		}

		socks = append(socks, sock)
	}

	return socks, nil
}

// sessionName derives a tmux-safe session name from a worktree map key (host/seg/.../last).
// Dots and colons are replaced with tmux-safe Unicode equivalents; slashes are preserved.
func sessionName(key string) string {
	return sessionNameReplacer.Replace(key)
}

// exactTarget escapes a tmux target given by *name* so that it matches only that
// exact name.
//
// tmux resolves an unescaped -t target by trying, in order: the exact name, a
// name prefix, then an fnmatch(3) pattern. Pane group names are
// derived from project IDs, so a project whose name is a prefix of another
// project's name on the same host (say "host/name" and "host/name-two") would
// otherwise resolve to whichever session happens to match first. Prefixing the
// target with "=" restricts tmux to exact matching.
//
// This is only for targets given by name. Targets that are tmux-assigned IDs
// (pane IDs of the form %N) are already unambiguous, and "=%1" is not valid
// target syntax — never pass those through here.
func exactTarget(name string) string {
	return "=" + name
}
