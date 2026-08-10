// Package story contains the `swm story` sub-commands.
package story

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"
	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
	"github.com/kalbasit/swm/cmd/swm/internal/hookexec"
)

// errUnexpectedPluginType is returned when a plugin client has an unexpected type.
var errUnexpectedPluginType = errors.New("unexpected plugin type")

// NewAttachCmd returns the `swm story attach` command. It ensures the project
// identified by the current working directory is attached to the resolved story,
// creating its worktree and running worktree hooks only when the project is not
// already attached. It performs no session/multiplexer work.
func NewAttachCmd(
	store coreStory.Store,
	mgr pluginManager,
	resolver *layout.Resolver,
	hooks hookexec.Runner,
	defaultStory string,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attach [<story-name>]",
		Short: "Ensure the current project's worktree exists for a story",
		Long: "Ensure the project in the current directory is attached to a story, " +
			"creating its worktree and running worktree hooks if it is not already " +
			"attached. Idempotent and safe to call blindly. Does not touch the " +
			"current multiplexer session.",
		Args: cobra.MaximumNArgs(1),
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			mgr.Warm(cmd.Context(), "vcs") //nolint:errcheck,gosec // Warm always returns nil; errors deferred to Get

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var name string
			if len(args) == 1 {
				name = args[0]
			} else {
				name = os.Getenv("SWM_STORY")
			}

			if name == "" {
				return errNoStoryName
			}

			return attachProject(cmd.Context(), cmd, name, store, mgr, resolver, hooks, defaultStory)
		},
	}

	cmd.ValidArgsFunction = storyNameCompletion(store)

	return cmd
}

// attachProject resolves the current project and ensures it is attached to the
// named story. See the workflow-commands spec, "swm story attach", for the full
// state machine (already-attached no-op, create, and reconcile paths).
func attachProject(
	ctx context.Context,
	cmd *cobra.Command,
	name string,
	store coreStory.Store,
	mgr pluginManager,
	resolver *layout.Resolver,
	hooks hookexec.Runner,
	defaultStory string,
) error {
	st, err := store.Get(ctx, name)
	if err != nil {
		if errors.Is(err, coreStory.ErrStoryNotFound) {
			return fmt.Errorf("%w: %s", coreStory.ErrStoryNotFound, name)
		}

		return fmt.Errorf("loading story %q: %w", name, err)
	}

	raw, err := mgr.Get(ctx, "vcs")
	if err != nil {
		return fmt.Errorf("loading vcs plugin: %w", err)
	}

	vcs, ok := raw.(pluginv1.VCSClient)
	if !ok {
		return fmt.Errorf("%w: %T", errUnexpectedPluginType, raw)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("determining working directory: %w", err)
	}

	pid, err := vcs.DetectProjectAtPath(ctx, &pluginv1.DetectAtPathRequest{Path: cwd})
	if err != nil {
		return fmt.Errorf("detecting project at %s: %w", cwd, err)
	}

	key := projectKey(pid.GetHost(), pid.GetSegments())

	// Already attached — nothing to do.
	if projectAttached(st, key) {
		cmd.Printf("project %s already attached to story %q\n", key, name)

		return nil
	}

	worktreePath := resolver.WorktreePath(name, pid)
	repoPath := resolver.CanonicalPath(pid)
	projectPath := strings.Join(pid.GetSegments(), "/")

	// Reconcile: a worktree already exists on disk but the store does not record
	// it (drift from a manually-created worktree). Attach in the store only —
	// nothing is being created, so create hooks do not run. The default story is
	// excluded because its worktree path is the always-present canonical checkout.
	if name != defaultStory && worktreeExists(worktreePath) {
		if err := attachToStore(ctx, store, st, pid); err != nil {
			return err
		}

		cmd.Printf("reconciled existing worktree for %s into story %q\n", key, name)

		return nil
	}

	// Create path: run hooks around worktree creation (workflow-commands spec,
	// swm workspace open, steps 6a-6d) minus any session work.
	if err := hooks.Run(ctx, hookexec.RunConfig{
		Event:        "pre-worktree-create",
		CodeRoot:     resolver.CodeRoot(),
		StoryName:    name,
		ProjectHost:  pid.GetHost(),
		ProjectPath:  projectPath,
		WorktreePath: worktreePath,
		RepoPath:     repoPath,
		WorkDir:      repoPath,
	}); err != nil {
		return fmt.Errorf("pre-worktree-create hook: %w", err)
	}

	if name != defaultStory {
		if _, err := vcs.CreateWorktree(ctx, &pluginv1.CreateWorktreeRequest{
			ProjectId:    pid,
			StoryName:    name,
			BranchName:   st.BranchName,
			RepoPath:     repoPath,
			WorktreePath: worktreePath,
		}); err != nil {
			return fmt.Errorf("creating worktree: %w", err)
		}
	}

	if err := attachToStore(ctx, store, st, pid); err != nil {
		return err
	}

	if err := hooks.Run(ctx, hookexec.RunConfig{
		Event:        "post-worktree-create",
		CodeRoot:     resolver.CodeRoot(),
		StoryName:    name,
		ProjectHost:  pid.GetHost(),
		ProjectPath:  projectPath,
		WorktreePath: worktreePath,
		RepoPath:     repoPath,
		WorkDir:      worktreePath,
	}); err != nil {
		slog.WarnContext(ctx, "post-worktree-create hook failed", "err", err)
	}

	cmd.Printf("attached %s to story %q\n", key, name)

	return nil
}

// attachToStore appends the project to the story and persists it. A concurrent
// attach that already recorded the project (ErrProjectAlreadyAttached) is
// treated as success so the command stays idempotent.
func attachToStore(
	ctx context.Context,
	store coreStory.Store,
	st *coreStory.Story,
	pid *pluginv1.ProjectID,
) error {
	st.Projects = append(st.Projects, coreStory.Project{
		Host:     pid.GetHost(),
		Segments: pid.GetSegments(),
	})

	if err := store.Update(ctx, st); err != nil {
		if errors.Is(err, coreStory.ErrProjectAlreadyAttached) {
			return nil
		}

		return fmt.Errorf("attaching project to story: %w", err)
	}

	return nil
}

// projectAttached reports whether the project identified by key is already
// listed in the story.
func projectAttached(st *coreStory.Story, key string) bool {
	for i := range st.Projects {
		p := &st.Projects[i]
		if projectKey(p.Host, p.Segments) == key {
			return true
		}
	}

	return false
}

// projectKey renders a project's canonical "host/seg1/.../segN" identity.
func projectKey(host string, segments []string) string {
	return host + "/" + strings.Join(segments, "/")
}

// worktreeExists reports whether a checkout already lives at worktreePath,
// detected by the presence of a .git entry (a file for a git worktree, a
// directory for a canonical clone).
func worktreeExists(worktreePath string) bool {
	if worktreePath == "" {
		return false
	}

	//nolint:gosec // worktreePath comes from the trusted host layout resolver; this only stats a .git entry
	_, err := os.Stat(filepath.Join(worktreePath, ".git"))

	return err == nil
}

// storyNameCompletion returns a cobra completion function listing all story
// names for a single positional argument.
func storyNameCompletion(
	store coreStory.Store,
) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		stories, err := store.List(cmd.Context())
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		names := make([]string, len(stories))
		for i, s := range stories {
			names[i] = s.Name
		}

		return names, cobra.ShellCompDirectiveNoFileComp
	}
}
