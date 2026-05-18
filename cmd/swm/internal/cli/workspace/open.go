// Package workspace contains the `swm workspace` sub-commands.
package workspace

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"
	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/cmd/swm/internal/config"
	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
	"github.com/kalbasit/swm/cmd/swm/internal/hookexec"
	"github.com/kalbasit/swm/cmd/swm/internal/termwidth"
)

// pluginManager is the subset of the CLI plugin manager used by this command.
type pluginManager interface {
	Get(ctx context.Context, capability string) (any, error)
}

// Sentinel errors.
var (
	errUnexpectedPluginType = errors.New("unexpected plugin type")
	errInvalidProjectKey    = errors.New("invalid project key: must be host/seg1/.../segN")
	errUnknownPickerKey     = errors.New("story picker returned unknown key")
)

// grpcStatuser is satisfied by any error that carries a gRPC status.
type grpcStatuser interface {
	GRPCStatus() *status.Status
}

// grpcCode unwraps the error chain to find a gRPC status code.
// status.Code() does not unwrap, so fmt.Errorf-wrapped gRPC errors
// would always return codes.Unknown without this helper.
func grpcCode(err error) codes.Code {
	if err == nil {
		return codes.OK
	}

	var s grpcStatuser
	if errors.As(err, &s) {
		return s.GRPCStatus().Code()
	}

	return codes.Unknown
}

// ExecFunc is the type used to replace the current process (default: syscall.Exec).
type ExecFunc func(argv0 string, argv []string, envv []string) error

// OpenOption configures NewOpenCmd behaviour.
type OpenOption func(*openCmdConfig)

type openCmdConfig struct {
	exec ExecFunc
}

// WithExecFunc injects an alternative to syscall.Exec. Intended for tests only.
func WithExecFunc(fn ExecFunc) OpenOption {
	return func(c *openCmdConfig) { c.exec = fn }
}

// NewOpenCmd returns the `swm workspace open` command.
func NewOpenCmd(
	cfg *config.Config,
	store coreStory.Store,
	mgr pluginManager,
	resolver *layout.Resolver,
	hooks hookexec.Runner,
	opts ...OpenOption,
) *cobra.Command {
	ocfg := &openCmdConfig{exec: syscall.Exec}
	for _, o := range opts {
		o(ocfg)
	}

	var killPane bool

	cmd := &cobra.Command{
		Use:   "open [story-name]",
		Short: "Open (or attach to) the workspace for a story",
		Long: "Open (or attach to) the workspace for a story. " +
			"If [story-name] is omitted, the command falls back to the $SWM_STORY " +
			"environment variable, and then to the default story configured in swm.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			var storyName string

			if len(args) > 0 {
				storyName = args[0]
			}

			if storyName == "" {
				storyName = os.Getenv("SWM_STORY")
			}

			// Attempt to load the picker plugin (optional — no error if absent).
			// Loaded early so it can be used for story selection before project selection.
			var pickerClient pluginv1.PickerClient

			if rawPicker, pickErr := mgr.Get(ctx, "picker"); pickErr == nil {
				if pc, ok := rawPicker.(pluginv1.PickerClient); ok {
					pickerClient = pc

					slog.DebugContext(ctx, "picker plugin loaded")
				}
			} else {
				slog.DebugContext(ctx, "picker plugin unavailable", "err", pickErr)
			}

			// If no story resolved yet and picker is available, show a story picker.
			if storyName == "" && pickerClient != nil {
				width := termwidth.Detect()

				selected, pickErr := pickStory(ctx, store, pickerClient, width)
				if pickErr != nil {
					code := grpcCode(pickErr)

					switch code { //nolint:exhaustive // default case handles all unexpected gRPC codes
					case codes.Aborted:
						// User cancelled the story picker — exit cleanly.
						return nil
					case codes.FailedPrecondition:
						// No TTY — fall through to default story.
						slog.DebugContext(ctx, "story picker unavailable (no TTY), using default story")
					default:
						return fmt.Errorf("story picker: %w", pickErr)
					}
				} else if selected != nil {
					storyName = selected.Name
				}
			}

			if storyName == "" {
				storyName = cfg.DefaultStory
			}

			st, err := store.Get(ctx, storyName)
			if err != nil {
				if errors.Is(err, coreStory.ErrStoryNotFound) {
					return fmt.Errorf("%w: %s", coreStory.ErrStoryNotFound, storyName)
				}

				return fmt.Errorf("loading story %q: %w", storyName, err)
			}

			slog.DebugContext(
				ctx, "workspace open",
				"story", storyName,
				"projects", len(st.Projects),
				"code_root", cfg.CodeRoot,
			)

			raw, err := mgr.Get(ctx, "session")
			if err != nil {
				return fmt.Errorf("loading session plugin: %w", err)
			}

			sess, ok := raw.(pluginv1.SessionClient)
			if !ok {
				return fmt.Errorf("%w: %T", errUnexpectedPluginType, raw)
			}

			if err := hooks.Run(ctx, hookexec.RunConfig{
				Event:     "pre-workspace-open",
				CodeRoot:  cfg.CodeRoot,
				StoryName: storyName,
				WorkDir:   cfg.CodeRoot,
			}); err != nil {
				return fmt.Errorf("pre-workspace-open hook: %w", err)
			}

			var openErr error
			if pickerClient != nil {
				openErr = openWithPicker(
					ctx, cmd, cfg, st, store, mgr, sess,
					pickerClient, resolver, hooks, storyName, killPane, ocfg.exec,
				)
				if openErr != nil {
					slog.DebugContext(
						ctx, "picker returned error, checking fallback",
						"code", grpcCode(openErr).String(),
						"err", openErr,
					)

					if grpcCode(openErr) == codes.FailedPrecondition {
						slog.DebugContext(ctx, "falling back to openAllAttached (no TTY)")
						openErr = openAllAttached(
							ctx, cmd, cfg, st, sess, resolver, hooks, storyName, killPane, ocfg.exec,
						)
					}
				}
			} else {
				openErr = openAllAttached(
					ctx, cmd, cfg, st, sess, resolver, hooks, storyName, killPane, ocfg.exec,
				)
			}

			return openErr
		},
	}

	cmd.Flags().BoolVar(&killPane, "kill-pane", false,
		"close the originating multiplexer pane after switching to the new workspace")

	return cmd
}

// buildSwitchToReq constructs a SwitchToRequest, adding origin-pane fields when
// killPane is true and the caller is identifiably inside a multiplexer session.
// The origin workspace ID is read directly from the host environment (not from the
// session plugin daemon, which may hold a stale environment). CurrentContext is
// consulted only for the pane-group safety guard: if the caller is already inside
// the target workspace and pane group the origin fields are left empty so the
// switch does not kill the current pane.
func buildSwitchToReq(
	ctx context.Context, sess pluginv1.SessionClient, wsID, pgID string, killPane bool,
) *pluginv1.SwitchToRequest {
	req := &pluginv1.SwitchToRequest{
		WorkspaceId: wsID,
		PaneGroupId: pgID,
	}

	if killPane {
		paneID, originWorkspaceID := detectMultiplexerOrigin()
		if paneID != "" {
			// Safety guard: avoid killing the origin pane when already in the target
			// workspace and pane group (the switch would be a no-op in that case).
			ctxResp, err := sess.CurrentContext(ctx, &pluginv1.Empty{})
			if err == nil && originWorkspaceID == wsID && ctxResp.GetPaneGroupId() == pgID {
				return req
			}

			req.CloseOriginWorkspaceId = originWorkspaceID
			req.CloseOriginPaneId = paneID
		}
	}

	return req
}

// detectMultiplexerOrigin returns the pane identifier and workspace identifier for
// the current multiplexer session by inspecting known environment variables. The
// workspace identifier is read from the host environment rather than from the
// session plugin daemon, which may hold a stale copy of the environment. Returns
// empty strings when no supported multiplexer is detected.
func detectMultiplexerOrigin() (paneID, workspaceID string) {
	// tmux: TMUX_PANE carries the pane reference; the socket path (workspace ID) is
	// the first comma-separated field of $TMUX ("<socket-path>,<pid>,<session-id>").
	if pane := os.Getenv("TMUX_PANE"); pane != "" {
		return pane, strings.SplitN(os.Getenv("TMUX"), ",", 2)[0]
	}
	// Zellij: ZELLIJ_PANE_ID is the pane identifier; ZELLIJ_SESSION_NAME is the workspace.
	if pane := os.Getenv("ZELLIJ_PANE_ID"); pane != "" {
		return pane, os.Getenv("ZELLIJ_SESSION_NAME")
	}

	return "", ""
}

// openWithPicker runs the interactive picker flow: enumerate all candidates, let the
// user pick one, lazily create the worktree if needed, then open a pane group.
func openWithPicker(
	ctx context.Context,
	cmd *cobra.Command,
	cfg *config.Config,
	st *coreStory.Story,
	store coreStory.Store,
	mgr pluginManager,
	sess pluginv1.SessionClient,
	pickerClient pluginv1.PickerClient,
	resolver *layout.Resolver,
	hooks hookexec.Runner,
	storyName string,
	killPane bool,
	execFn ExecFunc,
) error {
	// Build a deduplicated candidate set: attached projects + all on-disk repos.
	candidates := buildCandidates(cfg.CodeRoot, st, resolver)

	slog.DebugContext(
		ctx, "picker candidates built",
		"count", len(candidates),
		"code_root", cfg.CodeRoot,
		"story_projects", len(st.Projects),
	)

	stream, err := pickerClient.Pick(ctx)
	if err != nil {
		// Return unwrapped so the caller can inspect the gRPC status code.
		return err
	}

	for _, c := range candidates {
		if err := stream.Send(&pluginv1.PickItem{Key: c, Display: c}); err != nil {
			return fmt.Errorf("sending candidate to picker: %w", err)
		}
	}

	if err := stream.CloseSend(); err != nil {
		return fmt.Errorf("closing picker send: %w", err)
	}

	result, err := stream.Recv()
	if err != nil {
		slog.DebugContext(ctx, "picker recv", "code", status.Code(err).String(), "err", err)

		if status.Code(err) == codes.Aborted || errors.Is(err, io.EOF) {
			// User cancelled or no candidates — exit gracefully.
			return nil
		}

		return fmt.Errorf("receiving picker result: %w", err)
	}

	selectedKey := result.GetKey()

	// Derive the ProjectID from the selected key ("host/seg1/.../segN").
	pid, err := projectIDFromKey(selectedKey)
	if err != nil {
		return fmt.Errorf("parsing selected project key: %w", err)
	}

	worktreePath := resolver.WorktreePath(storyName, pid)

	// Check whether this project is already attached to the story.
	if !isAttached(st, selectedKey) {
		repoPath := resolver.CanonicalPath(pid)
		projectPath := strings.Join(pid.GetSegments(), "/")

		if err := hooks.Run(ctx, hookexec.RunConfig{
			Event:        "pre-worktree-create",
			CodeRoot:     cfg.CodeRoot,
			StoryName:    storyName,
			ProjectHost:  pid.GetHost(),
			ProjectPath:  projectPath,
			WorktreePath: worktreePath,
			RepoPath:     repoPath,
			WorkDir:      repoPath,
		}); err != nil {
			return fmt.Errorf("pre-worktree-create hook: %w", err)
		}

		if storyName != cfg.DefaultStory {
			rawVCS, err := mgr.Get(ctx, "vcs")
			if err != nil {
				return fmt.Errorf("loading vcs plugin: %w", err)
			}

			vcs, ok := rawVCS.(pluginv1.VCSClient)
			if !ok {
				return fmt.Errorf("%w: %T", errUnexpectedPluginType, rawVCS)
			}

			if _, err := vcs.CreateWorktree(ctx, &pluginv1.CreateWorktreeRequest{
				ProjectId:    pid,
				StoryName:    storyName,
				BranchName:   st.BranchName,
				RepoPath:     resolver.CanonicalPath(pid),
				WorktreePath: worktreePath,
			}); err != nil {
				return fmt.Errorf("creating worktree: %w", err)
			}
		}

		// Attach the project to the story store.
		st.Projects = append(st.Projects, coreStory.Project{
			Host:     pid.GetHost(),
			Segments: pid.GetSegments(),
		})

		if err := store.Update(ctx, st); err != nil {
			return fmt.Errorf("attaching project to story: %w", err)
		}

		_ = hooks.Run(ctx, hookexec.RunConfig{ //nolint:errcheck // post-* hooks always return nil; Run already logs failures
			Event:        "post-worktree-create",
			CodeRoot:     cfg.CodeRoot,
			StoryName:    storyName,
			ProjectHost:  pid.GetHost(),
			ProjectPath:  projectPath,
			WorktreePath: worktreePath,
			RepoPath:     repoPath,
			WorkDir:      worktreePath,
		})
	}

	// Ensure the workspace is open.
	ws, err := sess.OpenWorkspace(ctx, &pluginv1.OpenWorkspaceRequest{
		StoryName: storyName,
		WorktreePaths: map[string]string{
			selectedKey: worktreePath,
		},
	})
	if err != nil {
		return fmt.Errorf("opening workspace: %w", err)
	}

	pg, err := sess.OpenPaneGroup(ctx, &pluginv1.OpenPaneGroupRequest{
		WorkspaceId:  ws.GetWorkspaceId(),
		ProjectId:    pid,
		WorktreePath: worktreePath,
	})
	if err != nil {
		return fmt.Errorf("opening pane group: %w", err)
	}

	switchRes, err := sess.SwitchTo(ctx, buildSwitchToReq(ctx, sess, ws.GetWorkspaceId(), pg.GetPaneGroupId(), killPane))
	if err != nil {
		return fmt.Errorf("switching to pane group: %w", err)
	}

	cmd.Printf("opened pane group %q in workspace %q\n", pg.GetPaneGroupId(), storyName)

	// Run the post hook before exec so it is not skipped when the host process
	// is replaced by syscall.Exec.
	_ = hooks.Run(ctx, hookexec.RunConfig{ //nolint:errcheck // post-* hooks always return nil; Run already logs failures
		Event:     "post-workspace-open",
		CodeRoot:  cfg.CodeRoot,
		StoryName: storyName,
		WorkDir:   worktreePath,
	})

	if argv := switchRes.GetExecArgv(); len(argv) > 0 {
		if err := execFn(argv[0], argv, os.Environ()); err != nil {
			return fmt.Errorf("exec after switch: %w", err)
		}
	}

	return nil
}

// openAllAttached is the Phase 1 fallback: open a workspace with all attached projects.
func openAllAttached(
	ctx context.Context,
	cmd *cobra.Command,
	cfg *config.Config,
	st *coreStory.Story,
	sess pluginv1.SessionClient,
	resolver *layout.Resolver,
	hooks hookexec.Runner,
	storyName string,
	killPane bool,
	execFn ExecFunc,
) error {
	worktreePaths := make(map[string]string, len(st.Projects))

	for i := range st.Projects {
		p := &st.Projects[i]
		pid := &pluginv1.ProjectID{Host: p.Host, Segments: p.Segments}
		key := p.Host + "/" + strings.Join(p.Segments, "/")
		worktreePaths[key] = resolver.WorktreePath(storyName, pid)
	}

	ws, err := sess.OpenWorkspace(ctx, &pluginv1.OpenWorkspaceRequest{
		StoryName:     storyName,
		WorktreePaths: worktreePaths,
	})
	if err != nil {
		return fmt.Errorf("opening workspace: %w", err)
	}

	cmd.Printf("workspace opened for story %q\n", storyName)

	// With no attached projects there is no pane group to switch to.
	if len(st.Projects) == 0 {
		return nil
	}

	// Open a pane group for the first attached project and switch to it so that
	// exec (tmux attach-session) works consistently with the picker path.
	first := &st.Projects[0]
	firstPID := &pluginv1.ProjectID{Host: first.Host, Segments: first.Segments}
	firstKey := first.Host + "/" + strings.Join(first.Segments, "/")

	pg, err := sess.OpenPaneGroup(ctx, &pluginv1.OpenPaneGroupRequest{
		WorkspaceId:  ws.GetWorkspaceId(),
		ProjectId:    firstPID,
		WorktreePath: worktreePaths[firstKey],
	})
	if err != nil {
		return fmt.Errorf("opening pane group: %w", err)
	}

	// Run the post hook before exec so it is not skipped when the host process
	// is replaced by syscall.Exec.
	_ = hooks.Run(ctx, hookexec.RunConfig{ //nolint:errcheck // post-* hooks always return nil; Run already logs failures
		Event:     "post-workspace-open",
		CodeRoot:  cfg.CodeRoot,
		StoryName: storyName,
		WorkDir:   worktreePaths[firstKey],
	})

	switchRes, err := sess.SwitchTo(ctx, buildSwitchToReq(ctx, sess, ws.GetWorkspaceId(), pg.GetPaneGroupId(), killPane))
	if err != nil {
		return fmt.Errorf("switching to pane group: %w", err)
	}

	if argv := switchRes.GetExecArgv(); len(argv) > 0 {
		if err := execFn(argv[0], argv, os.Environ()); err != nil {
			return fmt.Errorf("exec after switch: %w", err)
		}
	}

	return nil
}

// buildCandidates returns a deduplicated list of project key strings,
// combining projects already attached to the story with all repositories on disk.
func buildCandidates(codeRoot string, st *coreStory.Story, resolver *layout.Resolver) []string {
	seen := make(map[string]struct{})

	var result []string

	addKey := func(key string) {
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			result = append(result, key)
		}
	}

	// Attached projects first so they appear at the top of the picker.
	for i := range st.Projects {
		p := &st.Projects[i]
		addKey(p.Host + "/" + strings.Join(p.Segments, "/"))
	}

	// All on-disk repositories.
	reposDir := filepath.Join(codeRoot, "repositories")

	slog.Default().Debug("scanning repos dir", "path", reposDir)

	//nolint:errcheck // walking the repos dir is best-effort; missing repos are simply excluded
	_ = filepath.WalkDir(reposDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // skip unreadable entries silently
		}

		if !d.IsDir() || d.Name() != ".git" {
			return nil
		}

		id := resolver.ProjectIDFromPath(filepath.Dir(path))
		if id == nil {
			return nil
		}

		addKey(id.Host + "/" + strings.Join(id.GetSegments(), "/"))

		return filepath.SkipDir
	})

	return result
}

// isAttached reports whether the project identified by key is already in the story.
func isAttached(st *coreStory.Story, key string) bool {
	for i := range st.Projects {
		p := &st.Projects[i]
		if p.Host+"/"+strings.Join(p.Segments, "/") == key {
			return true
		}
	}

	return false
}

// pickStory shows a story picker and returns the story the user selected.
// Errors are propagated as-is so the caller can inspect gRPC status codes
// (codes.Aborted = user cancelled; codes.FailedPrecondition = no TTY).
func pickStory(
	ctx context.Context,
	st coreStory.Store,
	pickerClient pluginv1.PickerClient,
	width int,
) (*coreStory.Story, error) {
	stories, err := st.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing stories for picker: %w", err)
	}

	sorted := SortStoriesForPicker(stories)

	stream, err := pickerClient.Pick(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	for _, s := range sorted {
		display := BuildStoryDisplay(s, width, now)
		if sendErr := stream.Send(&pluginv1.PickItem{Key: s.Name, Display: display}); sendErr != nil {
			return nil, fmt.Errorf("sending story to picker: %w", sendErr)
		}
	}

	if err := stream.CloseSend(); err != nil {
		return nil, fmt.Errorf("closing story picker send: %w", err)
	}

	result, err := stream.Recv()
	if err != nil {
		return nil, err // caller inspects gRPC status
	}

	selectedKey := result.GetKey()

	for _, s := range stories {
		if s.Name == selectedKey {
			return s, nil
		}
	}

	return nil, fmt.Errorf("%q: %w", selectedKey, errUnknownPickerKey)
}

// projectIDFromKey parses a "host/seg1/.../segN" string into a ProjectID.
func projectIDFromKey(key string) (*pluginv1.ProjectID, error) {
	parts := strings.SplitN(key, "/", 2)                    //nolint:mnd // split into host and the rest
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" { //nolint:mnd // need host and at least one segment
		return nil, fmt.Errorf("%q: %w", key, errInvalidProjectKey)
	}

	segments := strings.Split(parts[1], "/")
	if slices.Contains(segments, "") {
		return nil, fmt.Errorf("%q: %w (empty segment)", key, errInvalidProjectKey)
	}

	return &pluginv1.ProjectID{
		Host:     parts[0],
		Segments: segments,
	}, nil
}
