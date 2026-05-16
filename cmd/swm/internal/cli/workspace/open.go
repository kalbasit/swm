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

	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"
	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/cmd/swm/internal/config"
	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
	"github.com/kalbasit/swm/cmd/swm/internal/hookexec"
)

// pluginManager is the subset of the CLI plugin manager used by this command.
type pluginManager interface {
	Get(ctx context.Context, capability string) (any, error)
}

// Sentinel errors.
var (
	errUnexpectedPluginType = errors.New("unexpected plugin type")
	errInvalidProjectKey    = errors.New("invalid project key: must be host/seg1/.../segN")
)

// NewOpenCmd returns the `swm workspace open` command.
func NewOpenCmd(
	cfg *config.Config,
	store coreStory.Store,
	mgr pluginManager,
	resolver *layout.Resolver,
	hooks hookexec.Runner,
) *cobra.Command {
	var storyName string

	cmd := &cobra.Command{
		Use:   "open",
		Short: "Open (or attach to) the workspace for a story",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			if storyName == "" {
				storyName = os.Getenv("SWM_STORY")
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
			}); err != nil {
				return fmt.Errorf("pre-workspace-open hook: %w", err)
			}

			// Attempt to load the picker plugin (optional — no error if absent).
			var pickerClient pluginv1.PickerClient

			if rawPicker, pickErr := mgr.Get(ctx, "picker"); pickErr == nil {
				if pc, ok := rawPicker.(pluginv1.PickerClient); ok {
					pickerClient = pc
				}
			}

			var openErr error
			if pickerClient != nil {
				openErr = openWithPicker(ctx, cmd, cfg, st, store, mgr, sess, pickerClient, resolver, storyName)
				if openErr != nil && status.Code(openErr) == codes.FailedPrecondition {
					openErr = openAllAttached(ctx, cmd, st, sess, resolver, storyName)
				}
			} else {
				openErr = openAllAttached(ctx, cmd, st, sess, resolver, storyName)
			}

			if openErr != nil {
				return openErr
			}

			if err := hooks.Run(ctx, hookexec.RunConfig{
				Event:     "post-workspace-open",
				CodeRoot:  cfg.CodeRoot,
				StoryName: storyName,
			}); err != nil {
				slog.WarnContext(ctx, "post-workspace-open hook failed (ignored)", "err", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&storyName, "story", "", "story name (default: $SWM_STORY or default story)")

	return cmd
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
	storyName string,
) error {
	// Build a deduplicated candidate set: attached projects + all on-disk repos.
	candidates := buildCandidates(cfg.CodeRoot, st, resolver)

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
		if status.Code(err) == codes.Aborted || errors.Is(err, io.EOF) {
			// User cancelled — exit gracefully.
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

		// Attach the project to the story store.
		st.Projects = append(st.Projects, coreStory.Project{
			Host:     pid.GetHost(),
			Segments: pid.GetSegments(),
		})

		if err := store.Update(ctx, st); err != nil {
			return fmt.Errorf("attaching project to story: %w", err)
		}
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

	if _, err := sess.SwitchTo(ctx, &pluginv1.SwitchToRequest{
		WorkspaceId: ws.GetWorkspaceId(),
		PaneGroupId: pg.GetPaneGroupId(),
	}); err != nil {
		return fmt.Errorf("switching to pane group: %w", err)
	}

	cmd.Printf("opened pane group %q in workspace %q\n", pg.GetPaneGroupId(), storyName)

	return nil
}

// openAllAttached is the Phase 1 fallback: open a workspace with all attached projects.
func openAllAttached(
	ctx context.Context,
	cmd *cobra.Command,
	st *coreStory.Story,
	sess pluginv1.SessionClient,
	resolver *layout.Resolver,
	storyName string,
) error {
	worktreePaths := make(map[string]string, len(st.Projects))

	for i := range st.Projects {
		p := &st.Projects[i]
		pid := &pluginv1.ProjectID{Host: p.Host, Segments: p.Segments}
		key := p.Host + "/" + strings.Join(p.Segments, "/")
		worktreePaths[key] = resolver.WorktreePath(storyName, pid)
	}

	if _, err := sess.OpenWorkspace(ctx, &pluginv1.OpenWorkspaceRequest{
		StoryName:     storyName,
		WorktreePaths: worktreePaths,
	}); err != nil {
		return fmt.Errorf("opening workspace: %w", err)
	}

	cmd.Printf("workspace opened for story %q\n", storyName)

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
