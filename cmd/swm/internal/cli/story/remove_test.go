package story_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"

	"github.com/kalbasit/swm/cmd/swm/internal/cli/story"
	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
	"github.com/kalbasit/swm/cmd/swm/internal/hookexec"
)

func TestRemoveCmd_Force_NoProjects(t *testing.T) {
	t.Parallel()

	store := &stubStore{getStory: &coreStory.Story{Name: testStoryName}}
	mgr := &stubManager{}
	resolver := layout.NewResolver("/code")

	cmd := story.NewRemoveCmd(store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName, testForceFlag})

	require.NoError(t, cmd.Execute())
	require.True(t, store.deleted)
}

func TestRemoveCmd_NotFound(t *testing.T) {
	t.Parallel()

	store := &stubStore{getErr: coreStory.ErrStoryNotFound}
	mgr := &stubManager{}
	resolver := layout.NewResolver("/code")

	cmd := story.NewRemoveCmd(store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{"nonexistent", testForceFlag})

	require.Error(t, cmd.Execute())
}

func TestRemoveCmd_Force_WithProjects(t *testing.T) {
	t.Parallel()

	st := &coreStory.Story{
		Name: testStoryName,
		Projects: []coreStory.Project{
			{Host: testGitHubHost, Segments: []string{testKalbasitOrg, testSWMRepo}},
		},
	}
	store := &stubStore{getStory: st}
	vcs := &stubVCSClient{}
	mgr := &stubManager{vcs: vcs}
	resolver := layout.NewResolver("/code")

	cmd := story.NewRemoveCmd(store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName, testForceFlag})

	require.NoError(t, cmd.Execute())
	require.True(t, vcs.removeWorktreeCalled)
	require.True(t, store.deleted)
}

func TestRemoveCmd_HooksCalledInOrder(t *testing.T) {
	t.Parallel()

	st := &coreStory.Story{
		Name: testStoryName,
		Projects: []coreStory.Project{
			{Host: testGitHubHost, Segments: []string{testKalbasitOrg, testSWMRepo}},
		},
	}
	store := &stubStore{getStory: st}
	vcs := &stubVCSClient{}
	mgr := &stubManager{vcs: vcs}
	resolver := layout.NewResolver("/code")

	var called []string

	captureHook := hookexec.RunnerFunc(func(_ context.Context, cfg hookexec.RunConfig) error {
		called = append(called, cfg.Event)

		return nil
	})

	cmd := story.NewRemoveCmd(store, mgr, resolver, captureHook)
	cmd.SetArgs([]string{testStoryName, testForceFlag})

	require.NoError(t, cmd.Execute())
	require.Contains(t, called, "pre-story-remove")
	require.Contains(t, called, "pre-worktree-remove")
	require.Contains(t, called, "post-worktree-remove")
	require.Contains(t, called, "post-story-remove")
}
