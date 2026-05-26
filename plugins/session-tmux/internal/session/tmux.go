// Package session implements the swm Session capability using the system tmux binary.
package session

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/adrg/xdg"
	"github.com/pelletier/go-toml/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/plugins/session-tmux/internal/layout"
)

// buildVersion is set via -ldflags at build time.
var buildVersion = "dev" //nolint:gochecknoglobals // set via ldflags at link time

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
	sock := strings.SplitN(tmuxEnv, ",", 2)[0]
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
	sock := strings.SplitN(tmuxEnv, ",", 2)[0]
	inside := strings.HasPrefix(sock, t.socketDir)

	return &pluginv1.BoolValue{Value: inside}, nil
}

// ListWorkspaces streams all live swm tmux workspaces.
func (t *Tmux) ListWorkspaces(_ *pluginv1.Empty, stream pluginv1.Session_ListWorkspacesServer) error {
	entries, err := os.ReadDir(t.socketDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return status.Errorf(codes.Internal, "reading socket dir: %v", err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sock") {
			continue
		}

		sock := filepath.Join(t.socketDir, e.Name())

		// Probe liveness — skip dead sockets.
		if _, err := t.run(stream.Context(), "-S", sock, "list-sessions"); err != nil {
			continue
		}

		storyName := strings.TrimSuffix(e.Name(), ".sock")
		if err := stream.Send(&pluginv1.Workspace{
			WorkspaceId: sock,
			StoryName:   storyName,
		}); err != nil {
			return err
		}
	}

	return nil
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
	if _, err := t.run(ctx, "-S", sock, "has-session", "-t", name); err != nil {
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
		if _, err := t.run(ctx, "-S", sock, "switch-client", "-t", target); err != nil {
			return nil, err
		}

		resp = &pluginv1.SwitchToResponse{}
	} else {
		resp = &pluginv1.SwitchToResponse{
			ExecArgv: []string{t.tmuxBin, "-S", sock, "attach-session", "-t", target},
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

	if _, err := t.run(ctx, "-S", originSock, "kill-pane", "-t", paneID); err != nil {
		if isKillPaneNotFound(err) {
			return nil
		}

		return err
	}

	return nil
}

// isKillPaneNotFound reports whether a tmux kill-pane error indicates the pane
// or session no longer exists (expected race condition, safe to ignore).
func isKillPaneNotFound(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())

	return strings.Contains(msg, "no such pane") ||
		strings.Contains(msg, "can't find pane") ||
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

// sessionName derives a tmux-safe session name from a worktree map key (host/seg/.../last).
// Dots and colons are replaced with tmux-safe Unicode equivalents; slashes are preserved.
func sessionName(key string) string {
	return sessionNameReplacer.Replace(key)
}
