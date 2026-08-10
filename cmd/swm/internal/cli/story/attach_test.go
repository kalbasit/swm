package story_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"

	"github.com/kalbasit/swm/cmd/swm/internal/cli/story"
	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
	"github.com/kalbasit/swm/cmd/swm/internal/hookexec"
)

const defaultStoryName = "_default"

// recordingHooks records the events it is asked to run and can be configured to
// fail specific events.
type recordingHooks struct {
	mu     sync.Mutex
	events []string
	cfgs   map[string]hookexec.RunConfig
	errs   map[string]error
}

func (r *recordingHooks) Run(_ context.Context, cfg hookexec.RunConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.events = append(r.events, cfg.Event)

	if r.cfgs == nil {
		r.cfgs = make(map[string]hookexec.RunConfig)
	}

	r.cfgs[cfg.Event] = cfg

	return r.errs[cfg.Event]
}

func (r *recordingHooks) ran(event string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return slices.Contains(r.events, event)
}

// panicOnSessionManager fails the test if the session capability is ever loaded.
type panicOnSessionManager struct {
	*stubManager
}

func (m *panicOnSessionManager) Get(ctx context.Context, capability string) (any, error) {
	if capability == capSession {
		panic("story attach must not load the session plugin")
	}

	return m.stubManager.Get(ctx, capability)
}

func newAttachCmd(
	t *testing.T,
	store coreStory.Store,
	mgr *stubManager,
	resolver *layout.Resolver,
	hooks hookexec.Runner,
) *cobra.Command {
	t.Helper()

	cmd := story.NewAttachCmd(store, mgr, resolver, hooks, defaultStoryName)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	return cmd
}

func swmProjectStory(name string) *coreStory.Story {
	return &coreStory.Story{Name: name, BranchName: "feat/" + name}
}

func TestAttachCmd_PreRunE_WarmsVCSOnly(t *testing.T) {
	t.Setenv("SWM_STORY", "")

	store := &stubStore{getStory: swmProjectStory(testStoryName)}
	resolver := layout.NewResolver(t.TempDir(), defaultStoryName)
	rec := &warmRecordingManager{stubManager: &stubManager{vcs: &stubVCSClient{}}}

	cmd := story.NewAttachCmd(store, rec, resolver, hookexec.Noop, defaultStoryName)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute())
	require.Equal(t, []string{capVCS}, rec.warmedCaps, "story attach must warm only vcs (no session)")
}

func TestAttachCmd_NoArg_NoEnv_Errors(t *testing.T) {
	t.Setenv("SWM_STORY", "")

	store := &stubStore{}
	resolver := layout.NewResolver(t.TempDir(), defaultStoryName)
	mgr := &stubManager{vcs: &stubVCSClient{}}

	cmd := newAttachCmd(t, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{})

	require.Error(t, cmd.Execute())
	require.Empty(t, store.lastGetName, "no work must happen when no story is resolved")
}

func TestAttachCmd_ResolvesFromEnv(t *testing.T) {
	t.Setenv("SWM_STORY", testStoryName)

	store := &stubStore{getStory: swmProjectStory(testStoryName)}
	resolver := layout.NewResolver(t.TempDir(), defaultStoryName)
	mgr := &stubManager{vcs: &stubVCSClient{}}

	cmd := newAttachCmd(t, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{})

	require.NoError(t, cmd.Execute())
	require.Equal(t, testStoryName, store.lastGetName)
}

func TestAttachCmd_ArgOverridesEnv(t *testing.T) {
	t.Setenv("SWM_STORY", testStoryName)

	store := &stubStore{getStory: swmProjectStory("other-story")}
	resolver := layout.NewResolver(t.TempDir(), defaultStoryName)
	mgr := &stubManager{vcs: &stubVCSClient{}}

	cmd := newAttachCmd(t, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{"other-story"})

	require.NoError(t, cmd.Execute())
	require.Equal(t, "other-story", store.lastGetName)
}

func TestAttachCmd_StoryNotFound(t *testing.T) {
	t.Setenv("SWM_STORY", "")

	store := &stubStore{getErr: coreStory.ErrStoryNotFound}
	resolver := layout.NewResolver(t.TempDir(), defaultStoryName)
	vcs := &stubVCSClient{}
	mgr := &stubManager{vcs: vcs}

	cmd := newAttachCmd(t, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{"nonexistent"})

	require.ErrorIs(t, cmd.Execute(), coreStory.ErrStoryNotFound)
	require.False(t, vcs.createWorktreeCalled)
}

func TestAttachCmd_NotInsideRepository(t *testing.T) {
	t.Setenv("SWM_STORY", "")

	store := &stubStore{getStory: swmProjectStory(testStoryName)}
	resolver := layout.NewResolver(t.TempDir(), defaultStoryName)
	vcs := &stubVCSClient{detectErr: errNotFound}
	mgr := &stubManager{vcs: vcs}

	cmd := newAttachCmd(t, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName})

	require.Error(t, cmd.Execute())
	require.False(t, store.updateCalled, "no store write when the project cannot be detected")
	require.False(t, vcs.createWorktreeCalled)
}

func TestAttachCmd_DetectsProjectFromCwd(t *testing.T) {
	t.Setenv("SWM_STORY", "")

	store := &stubStore{getStory: swmProjectStory(testStoryName)}
	resolver := layout.NewResolver(t.TempDir(), defaultStoryName)
	vcs := &stubVCSClient{}
	mgr := &stubManager{vcs: vcs}

	cmd := newAttachCmd(t, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute())

	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.True(t, vcs.detectCalled)
	require.Equal(t, cwd, vcs.detectPath, "DetectProjectAtPath must be called with the working directory")
}

func TestAttachCmd_AlreadyAttached_Noop(t *testing.T) {
	t.Setenv("SWM_STORY", "")

	st := swmProjectStory(testStoryName)
	st.Projects = []coreStory.Project{{Host: testGitHubHost, Segments: []string{testKalbasitOrg, testSWMRepo}}}

	store := &stubStore{getStory: st}
	resolver := layout.NewResolver(t.TempDir(), defaultStoryName)
	vcs := &stubVCSClient{}
	mgr := &stubManager{vcs: vcs}
	hooks := &recordingHooks{}

	cmd := newAttachCmd(t, store, mgr, resolver, hooks)
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute())
	require.False(t, vcs.createWorktreeCalled, "no worktree creation for an attached project")
	require.False(t, store.updateCalled, "no store write for an attached project")
	require.Empty(t, hooks.events, "no hooks run for an attached project")
}

func TestAttachCmd_CreatesWorktree(t *testing.T) {
	t.Setenv("SWM_STORY", "")

	codeRoot := t.TempDir()
	store := &stubStore{getStory: swmProjectStory(testStoryName)}
	resolver := layout.NewResolver(codeRoot, defaultStoryName)
	vcs := &stubVCSClient{}
	mgr := &stubManager{vcs: vcs}
	hooks := &recordingHooks{}

	cmd := newAttachCmd(t, store, mgr, resolver, hooks)
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute())

	require.True(t, vcs.createWorktreeCalled)
	require.Equal(t, "feat/"+testStoryName, vcs.createWorktreeReq.GetBranchName())
	require.Equal(t, testGitHubHost, vcs.createWorktreeReq.GetProjectId().GetHost())

	require.True(t, store.updateCalled)
	require.Len(t, store.updatedStory.Projects, 1)
	require.Equal(t, testGitHubHost, store.updatedStory.Projects[0].Host)

	// Hook ordering and context.
	require.True(t, hooks.ran("pre-worktree-create"))
	require.True(t, hooks.ran("post-worktree-create"))

	wantWorktree := filepath.Join(codeRoot, "stories", testStoryName, testGitHubHost, testKalbasitOrg, testSWMRepo)
	require.Equal(t, wantWorktree, hooks.cfgs["pre-worktree-create"].WorktreePath)
	require.Equal(t, testGitHubHost, hooks.cfgs["pre-worktree-create"].ProjectHost)
	require.Equal(t, testKalbasitOrg+"/"+testSWMRepo, hooks.cfgs["pre-worktree-create"].ProjectPath)
}

func TestAttachCmd_Reconcile_WorktreeExists(t *testing.T) {
	t.Setenv("SWM_STORY", "")

	codeRoot := t.TempDir()
	worktree := filepath.Join(codeRoot, "stories", testStoryName, testGitHubHost, testKalbasitOrg, testSWMRepo)
	require.NoError(t, os.MkdirAll(worktree, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(worktree, ".git"), []byte("gitdir: whatever\n"), 0o600))

	store := &stubStore{getStory: swmProjectStory(testStoryName)}
	resolver := layout.NewResolver(codeRoot, defaultStoryName)
	vcs := &stubVCSClient{}
	mgr := &stubManager{vcs: vcs}
	hooks := &recordingHooks{}

	cmd := newAttachCmd(t, store, mgr, resolver, hooks)
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute())

	require.False(t, vcs.createWorktreeCalled, "must not re-create an existing worktree")
	require.Empty(t, hooks.events, "reconcile must not run create hooks")
	require.True(t, store.updateCalled, "reconcile must record the project in the store")
	require.Len(t, store.updatedStory.Projects, 1)
}

func TestAttachCmd_DefaultStory_SkipsCreateWorktree(t *testing.T) {
	t.Setenv("SWM_STORY", "")

	store := &stubStore{getStory: &coreStory.Story{Name: defaultStoryName}}
	resolver := layout.NewResolver(t.TempDir(), defaultStoryName)
	vcs := &stubVCSClient{}
	mgr := &stubManager{vcs: vcs}
	hooks := &recordingHooks{}

	cmd := newAttachCmd(t, store, mgr, resolver, hooks)
	cmd.SetArgs([]string{defaultStoryName})

	require.NoError(t, cmd.Execute())

	require.False(t, vcs.createWorktreeCalled, "default story uses the canonical checkout")
	require.True(t, hooks.ran("pre-worktree-create"))
	require.True(t, hooks.ran("post-worktree-create"))
	require.True(t, store.updateCalled)
}

func TestAttachCmd_PreHookAborts(t *testing.T) {
	t.Setenv("SWM_STORY", "")

	store := &stubStore{getStory: swmProjectStory(testStoryName)}
	resolver := layout.NewResolver(t.TempDir(), defaultStoryName)
	vcs := &stubVCSClient{}
	mgr := &stubManager{vcs: vcs}
	hooks := &recordingHooks{errs: map[string]error{"pre-worktree-create": errNotFound}}

	cmd := newAttachCmd(t, store, mgr, resolver, hooks)
	cmd.SetArgs([]string{testStoryName})

	require.Error(t, cmd.Execute())
	require.False(t, vcs.createWorktreeCalled, "pre-hook failure must prevent worktree creation")
	require.False(t, store.updateCalled, "pre-hook failure must prevent attach")
}

func TestAttachCmd_PostHookFails_Succeeds(t *testing.T) {
	t.Setenv("SWM_STORY", "")

	store := &stubStore{getStory: swmProjectStory(testStoryName)}
	resolver := layout.NewResolver(t.TempDir(), defaultStoryName)
	vcs := &stubVCSClient{}
	mgr := &stubManager{vcs: vcs}
	hooks := &recordingHooks{errs: map[string]error{"post-worktree-create": errNotFound}}

	cmd := newAttachCmd(t, store, mgr, resolver, hooks)
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute(), "post-hook failure must not fail the command")
	require.True(t, store.updateCalled, "project must remain attached")
}

func TestAttachCmd_ConcurrentAttach_TreatedAsSuccess(t *testing.T) {
	t.Setenv("SWM_STORY", "")

	store := &stubStore{
		getStory:  swmProjectStory(testStoryName),
		updateErr: coreStory.ErrProjectAlreadyAttached,
	}
	resolver := layout.NewResolver(t.TempDir(), defaultStoryName)
	vcs := &stubVCSClient{}
	mgr := &stubManager{vcs: vcs}

	cmd := newAttachCmd(t, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute(), "a lost attach race must be treated as success")
}

func TestAttachCmd_NoSessionLoaded(t *testing.T) {
	t.Setenv("SWM_STORY", "")

	store := &stubStore{getStory: swmProjectStory(testStoryName)}
	resolver := layout.NewResolver(t.TempDir(), defaultStoryName)
	mgr := &panicOnSessionManager{stubManager: &stubManager{vcs: &stubVCSClient{}}}

	cmd := story.NewAttachCmd(store, mgr, resolver, hookexec.Noop, defaultStoryName)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute())
}

func TestAttachCmd_Completion_ListsStoryNames(t *testing.T) {
	t.Parallel()

	store := &stubStore{listStories: []*coreStory.Story{{Name: "feat-a"}, {Name: "feat-b"}}}
	resolver := layout.NewResolver(t.TempDir(), defaultStoryName)
	mgr := &stubManager{vcs: &stubVCSClient{}}

	cmd := story.NewAttachCmd(store, mgr, resolver, hookexec.Noop, defaultStoryName)

	names, directive := cmd.ValidArgsFunction(cmd, nil, "")
	require.Equal(t, []string{"feat-a", "feat-b"}, names)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}
