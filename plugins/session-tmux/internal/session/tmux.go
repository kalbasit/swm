// Package session implements the swm Session capability using the system tmux binary.
package session

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"
)

// buildVersion is set via -ldflags at build time.
var buildVersion = "dev" //nolint:gochecknoglobals // set via ldflags at link time

// Tmux implements pluginv1.SessionServer by shelling out to the system tmux.
type Tmux struct {
	tmuxBin   string
	socketDir string
}

// New returns a Tmux instance using the system tmux binary.
func New() (*Tmux, error) {
	bin, err := exec.LookPath("tmux")
	if err != nil {
		return nil, fmt.Errorf("tmux binary not found in PATH: %w", err)
	}

	return &Tmux{
		tmuxBin:   bin,
		socketDir: filepath.Join(xdg.RuntimeDir, "swm", "tmux"),
	}, nil
}

// NewWithBin returns a Tmux instance with an injected binary path and socket dir (for tests).
func NewWithBin(tmuxBin, socketDir string) *Tmux {
	return &Tmux{tmuxBin: tmuxBin, socketDir: socketDir}
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
			Name:    "tmux",
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
	segments := req.GetProjectId().GetSegments()

	if len(segments) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "project_id has no segments")
	}

	name := segments[len(segments)-1]

	// Create session if it doesn't exist yet.
	if _, err := t.run(ctx, "-S", sock, "has-session", "-t", name); err != nil {
		if _, err := t.run(ctx, "-S", sock, "new-session", "-d", "-s", name, "-c", req.GetWorktreePath()); err != nil {
			return nil, err
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
func (t *Tmux) OpenWorkspace(ctx context.Context, req *pluginv1.OpenWorkspaceRequest) (*pluginv1.Workspace, error) {
	sock := t.socketPath(req.GetStoryName())

	if err := os.MkdirAll(filepath.Dir(sock), 0o700); err != nil {
		return nil, status.Errorf(codes.Internal, "creating socket dir: %v", err)
	}

	// Probe whether the tmux server is already running.
	if _, err := t.run(ctx, "-S", sock, "list-sessions"); err != nil {
		// Server not running — start it with the first worktree as the initial session.
		var firstName, firstPath string
		for key, path := range req.GetWorktreePaths() {
			firstName = sessionName(key)
			firstPath = path

			break
		}

		if _, err := t.run(ctx, "-S", sock, "new-session", "-d", "-s", firstName, "-c", firstPath); err != nil {
			return nil, err
		}
	}

	// Ensure a session exists for every worktree path.
	for key, path := range req.GetWorktreePaths() {
		name := sessionName(key)
		if _, err := t.run(ctx, "-S", sock, "has-session", "-t", name); err != nil {
			// Session absent — create it.
			if _, err := t.run(ctx, "-S", sock, "new-session", "-d", "-s", name, "-c", path); err != nil {
				return nil, err
			}
		}
	}

	return &pluginv1.Workspace{
		WorkspaceId: sock,
		StoryName:   req.GetStoryName(),
	}, nil
}

// SwitchTo brings the given pane group into focus.
func (t *Tmux) SwitchTo(ctx context.Context, req *pluginv1.SwitchToRequest) (*pluginv1.Empty, error) {
	sock := req.GetWorkspaceId()
	target := req.GetPaneGroupId()

	var err error
	if os.Getenv("TMUX") != "" {
		_, err = t.run(ctx, "-S", sock, "switch-client", "-t", target)
	} else {
		_, err = t.run(ctx, "-S", sock, "attach-session", "-t", target)
	}

	if err != nil {
		return nil, err
	}

	return &pluginv1.Empty{}, nil
}

func (t *Tmux) run(ctx context.Context, args ...string) (string, error) {
	var stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, t.tmuxBin, args...) //nolint:gosec // tmuxBin from LookPath, args are controlled
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

// sessionName derives a tmux session name from a worktree map key (host/seg/.../last).
func sessionName(key string) string {
	parts := strings.Split(key, "/")

	return parts[len(parts)-1]
}
