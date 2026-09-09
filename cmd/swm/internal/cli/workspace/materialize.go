package workspace

import (
	"context"
	"fmt"
	"strings"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"
	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
)

// Hook events this package runs. Named once so `open` and `ensure` cannot
// drift onto different strings for the same event.
const (
	eventPreWorkspaceOpen  = "pre-workspace-open"
	eventPostWorkspaceOpen = "post-workspace-open"
)

// openedGroup is one pane group that was opened, paired with the project key it
// belongs to so a caller can find the group for a project it cares about
// without re-deriving the key.
type openedGroup struct {
	ProjectKey  string
	PaneGroupID string
}

// openedWorkspace is what a story's workspace looks like once it exists: its
// id, the worktree path of every attached project, and the pane groups that
// were opened in it.
type openedWorkspace struct {
	ID            string
	WorktreePaths map[string]string
	Groups        []openedGroup
}

// worktreePathsFor derives the worktree path of every project attached to the
// story, keyed by project key.
func worktreePathsFor(resolver *layout.Resolver, st *coreStory.Story, storyName string) map[string]string {
	paths := make(map[string]string, len(st.Projects))

	for i := range st.Projects {
		p := &st.Projects[i]
		pid := &pluginv1.ProjectID{Host: p.Host, Segments: p.Segments}
		paths[projectKey(p)] = resolver.WorktreePath(storyName, pid)
	}

	return paths
}

// projectKey renders a project as the host/segments string used everywhere a
// project is named outside the multiplexer.
func projectKey(p *coreStory.Project) string {
	return p.Host + "/" + strings.Join(p.Segments, "/")
}

// openWorkspaceWithGroups opens the story's workspace and a pane group for each
// project in groupsFor, and reports what it opened.
//
// It deliberately knows nothing about switching to a pane group, replacing the
// process, or closing the pane it was called from. `workspace ensure` must be
// unable to do any of those, so they stay with the caller that wants them
// rather than sitting behind a flag here.
//
// groupsFor is passed rather than assumed: `open` wants one group because it is
// about to put the cursor in it, and `ensure` wants every project's group
// because a caller asking for a story's workspace has no cursor and cannot know
// which project the next command will name.
func openWorkspaceWithGroups(
	ctx context.Context,
	sess pluginv1.SessionClient,
	resolver *layout.Resolver,
	st *coreStory.Story,
	storyName string,
	groupsFor []coreStory.Project,
) (*openedWorkspace, error) {
	worktreePaths := worktreePathsFor(resolver, st, storyName)

	ws, err := sess.OpenWorkspace(ctx, &pluginv1.OpenWorkspaceRequest{
		StoryName:     storyName,
		WorktreePaths: worktreePaths,
	})
	if err != nil {
		return nil, fmt.Errorf("opening workspace: %w", err)
	}

	opened := &openedWorkspace{
		ID:            ws.GetWorkspaceId(),
		WorktreePaths: worktreePaths,
		Groups:        make([]openedGroup, 0, len(groupsFor)),
	}

	for i := range groupsFor {
		p := &groupsFor[i]
		key := projectKey(p)

		pg, err := sess.OpenPaneGroup(ctx, &pluginv1.OpenPaneGroupRequest{
			WorkspaceId:  opened.ID,
			ProjectId:    &pluginv1.ProjectID{Host: p.Host, Segments: p.Segments},
			WorktreePath: worktreePaths[key],
		})
		if err != nil {
			return nil, fmt.Errorf("opening pane group for %s: %w", key, err)
		}

		opened.Groups = append(opened.Groups, openedGroup{
			ProjectKey:  key,
			PaneGroupID: pg.GetPaneGroupId(),
		})
	}

	return opened, nil
}
