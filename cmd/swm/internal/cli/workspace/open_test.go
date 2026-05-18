package workspace_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"
	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/cmd/swm/internal/cli/workspace"
	"github.com/kalbasit/swm/cmd/swm/internal/config"
	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
	"github.com/kalbasit/swm/cmd/swm/internal/hookexec"
)

const (
	testCodeRoot     = "/code"
	testDefaultStory = "_default"
	testStoryName    = "feat-x"
	testBranchName   = "feat/feat-x"
	testHost         = "github.com"
	testOwner        = "kalbasit"
	testSegment      = "swm"

	eventPreWorktreeCreate  = "pre-worktree-create"
	eventPostWorktreeCreate = "post-worktree-create"

	flagKillPane = "--kill-pane"
)

var (
	errNoPlugin = errors.New("no plugin")
	errFakeHook = errors.New("hook error")
)

func TestOpenCmd_WithPositionalArg(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{Name: testStoryName}}
	sess := &stubSess{}
	mgr := &stubMgr{sess: sess}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute())
	require.Equal(t, testStoryName, sess.lastOpenReq.GetStoryName())
}

func TestOpenCmd_PositionalArgOverridesEnv(t *testing.T) {
	t.Setenv("SWM_STORY", "env-story")

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{Name: testStoryName}}
	sess := &stubSess{}
	mgr := &stubMgr{sess: sess}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute())
	require.Equal(t, testStoryName, sess.lastOpenReq.GetStoryName())
}

func TestOpenCmd_DefaultStory(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{Name: testDefaultStory}}
	sess := &stubSess{}
	mgr := &stubMgr{sess: sess}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{})

	require.NoError(t, cmd.Execute())
	require.Equal(t, testDefaultStory, sess.lastOpenReq.GetStoryName())
}

func TestOpenCmd_StoryNotFound(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getErr: coreStory.ErrStoryNotFound}
	mgr := &stubMgr{}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{"nonexistent"})

	require.Error(t, cmd.Execute())
}

func TestOpenCmd_NoProjects(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{Name: testStoryName, Projects: nil}}
	sess := &stubSess{}
	mgr := &stubMgr{sess: sess}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute())
	require.Empty(t, sess.lastOpenReq.GetWorktreePaths())
}

func TestOpenCmd_WithPicker_ProjectAlreadyAttached(t *testing.T) {
	t.Parallel()

	const selectedKey = "github.com/kalbasit/swm"

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{
		Name: testStoryName,
		Projects: []coreStory.Project{
			{Host: testHost, Segments: []string{testOwner, testSegment}},
		},
	}}
	sess := &stubSess{}
	vcs := &stubVCS{}
	picker := &stubPickerClient{selectedKey: selectedKey}
	mgr := &stubMgr{sess: sess, vcs: vcs, picker: picker}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute())

	// Already attached — no CreateWorktree call.
	require.False(t, vcs.createCalled)

	// Pane group was opened.
	require.NotNil(t, sess.lastPaneGroupReq)
	require.Equal(t, selectedKey, sess.lastPaneGroupReq.GetProjectId().GetHost()+"/"+
		"kalbasit/"+testSegment)
}

func TestOpenCmd_WithPicker_ProjectNotAttached(t *testing.T) {
	t.Parallel()

	const selectedKey = "github.com/kalbasit/dotfiles"

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{
		Name:       testStoryName,
		BranchName: testBranchName,
	}}
	sess := &stubSess{}
	vcs := &stubVCS{}
	picker := &stubPickerClient{selectedKey: selectedKey}
	mgr := &stubMgr{sess: sess, vcs: vcs, picker: picker}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute())

	// Not attached — CreateWorktree must be called.
	require.True(t, vcs.createCalled)

	// Story store updated.
	require.True(t, store.updateCalled)

	// Pane group was opened.
	require.NotNil(t, sess.lastPaneGroupReq)
}

func TestOpenCmd_WithPicker_PreWorktreeCreateHookAborts(t *testing.T) {
	t.Parallel()

	const selectedKey = "github.com/kalbasit/dotfiles"

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{
		Name:       testStoryName,
		BranchName: testBranchName,
	}}
	sess := &stubSess{}
	vcs := &stubVCS{}
	picker := &stubPickerClient{selectedKey: selectedKey}
	mgr := &stubMgr{sess: sess, vcs: vcs, picker: picker}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	hooks := hookexec.RunnerFunc(func(_ context.Context, rc hookexec.RunConfig) error {
		if rc.Event == eventPreWorktreeCreate {
			return errFakeHook
		}

		return nil
	})

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hooks)
	cmd.SetArgs([]string{testStoryName})

	require.Error(t, cmd.Execute())

	// Hook aborted — CreateWorktree must NOT have been called.
	require.False(t, vcs.createCalled)
}

func TestOpenCmd_WithPicker_PostWorktreeCreateHookFailureContinues(t *testing.T) {
	t.Parallel()

	const selectedKey = "github.com/kalbasit/dotfiles"

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{
		Name:       testStoryName,
		BranchName: testBranchName,
	}}
	sess := &stubSess{}
	vcs := &stubVCS{}
	picker := &stubPickerClient{selectedKey: selectedKey}
	mgr := &stubMgr{sess: sess, vcs: vcs, picker: picker}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	var postHookCalled bool

	hooks := hookexec.RunnerFunc(func(_ context.Context, rc hookexec.RunConfig) error {
		if rc.Event == eventPostWorktreeCreate {
			postHookCalled = true

			return errFakeHook
		}

		return nil
	})

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hooks)
	cmd.SetArgs([]string{testStoryName})

	// Post-hook failure must not abort the command.
	require.NoError(t, cmd.Execute())

	// Hook must have been invoked despite its failure being ignored.
	require.True(t, postHookCalled)

	// Workspace was opened.
	require.NotNil(t, sess.lastOpenReq)
}

func TestOpenCmd_WithPicker_PostWorktreeCreateHookReceivesContext(t *testing.T) {
	t.Parallel()

	const selectedKey = "github.com/kalbasit/dotfiles"

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{
		Name:       testStoryName,
		BranchName: testBranchName,
	}}
	sess := &stubSess{}
	vcs := &stubVCS{}
	picker := &stubPickerClient{selectedKey: selectedKey}
	mgr := &stubMgr{sess: sess, vcs: vcs, picker: picker}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	var capturedCfg hookexec.RunConfig

	hooks := hookexec.RunnerFunc(func(_ context.Context, rc hookexec.RunConfig) error {
		if rc.Event == eventPostWorktreeCreate {
			capturedCfg = rc
		}

		return nil
	})

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hooks)
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute())

	require.Equal(t, "github.com", capturedCfg.ProjectHost)
	require.Equal(t, "kalbasit/dotfiles", capturedCfg.ProjectPath)
	require.Equal(t, "/code/stories/feat-x/github.com/kalbasit/dotfiles", capturedCfg.WorktreePath)
	require.Equal(t, "/code/repositories/github.com/kalbasit/dotfiles", capturedCfg.RepoPath)
}

func TestOpenCmd_WithPicker_PostWorktreeCreateHookWorkDir(t *testing.T) {
	t.Parallel()

	const selectedKey = "github.com/kalbasit/dotfiles"

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{
		Name:       testStoryName,
		BranchName: testBranchName,
	}}
	sess := &stubSess{}
	vcs := &stubVCS{}
	picker := &stubPickerClient{selectedKey: selectedKey}
	mgr := &stubMgr{sess: sess, vcs: vcs, picker: picker}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	var capturedCfgs []hookexec.RunConfig

	hooks := hookexec.RunnerFunc(func(_ context.Context, rc hookexec.RunConfig) error {
		capturedCfgs = append(capturedCfgs, rc)

		return nil
	})

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hooks)
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute())

	for _, rc := range capturedCfgs {
		switch rc.Event {
		case eventPostWorktreeCreate:
			require.Equal(t, rc.WorktreePath, rc.WorkDir, "post-worktree-create WorkDir must be worktree path")
		case eventPreWorktreeCreate:
			require.Equal(t, rc.RepoPath, rc.WorkDir, "pre-worktree-create WorkDir must be repo path")
		}
	}
}

func TestOpenCmd_WithPicker_Cancelled(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{Name: testStoryName}}
	sess := &stubSess{}
	vcs := &stubVCS{}
	// Picker returns Aborted (user pressed Escape).
	picker := &stubPickerClient{cancelOnRecv: true}
	mgr := &stubMgr{sess: sess, vcs: vcs, picker: picker}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName})

	// Cancellation is not an error.
	require.NoError(t, cmd.Execute())
	require.Nil(t, sess.lastPaneGroupReq)
}

func TestOpenCmd_WithPicker_FailedPrecondition_FallsBack(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{
		Name: testStoryName,
		Projects: []coreStory.Project{
			{Host: testHost, Segments: []string{testOwner, testSegment}},
		},
	}}
	sess := &stubSess{}
	// Picker returns FailedPrecondition (e.g. no TTY available).
	picker := &stubPickerClient{pickErr: status.Error(codes.FailedPrecondition, "no tty")}
	mgr := &stubMgr{sess: sess, picker: picker}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName})

	// Should succeed by falling back to Phase-1 behavior.
	require.NoError(t, cmd.Execute())

	// Fallback path: OpenWorkspace AND OpenPaneGroup are both called.
	require.NotNil(t, sess.lastOpenReq, "expected OpenWorkspace to be called as fallback")
	require.NotNil(t, sess.lastPaneGroupReq, "expected OpenPaneGroup to be called in fallback path")
}

// TestOpenCmd_WithPicker_RecvFailedPrecondition_FallsBack covers the case where the
// picker's stream.Recv() returns FailedPrecondition (e.g. /dev/tty unavailable inside
// the handler, after all candidates have been received). The host must still fall back
// to openAllAttached, not surface the gRPC error to the caller.
func TestOpenCmd_WithPicker_RecvFailedPrecondition_FallsBack(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{
		Name: testStoryName,
		Projects: []coreStory.Project{
			{Host: testHost, Segments: []string{testOwner, testSegment}},
		},
	}}
	sess := &stubSess{}
	// Picker stream opens fine but Recv() returns FailedPrecondition (no TTY).
	picker := &stubPickerClient{recvErr: status.Error(codes.FailedPrecondition, "no tty")}
	mgr := &stubMgr{sess: sess, picker: picker}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute())
	require.NotNil(t, sess.lastOpenReq, "expected OpenWorkspace to be called as fallback")
	require.NotNil(t, sess.lastPaneGroupReq, "expected OpenPaneGroup to be called in fallback path")
}

func TestOpenCmd_WithPicker_InvalidKey_EmptyHost(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{Name: testStoryName}}
	sess := &stubSess{}
	picker := &stubPickerClient{selectedKey: "/seg1"} // empty host
	mgr := &stubMgr{sess: sess, picker: picker}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName})

	err := cmd.Execute()
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid project key")
}

func TestOpenCmd_WithPicker_InvalidKey_EmptySegments(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{Name: testStoryName}}
	sess := &stubSess{}
	picker := &stubPickerClient{selectedKey: "github.com/"} // empty segments
	mgr := &stubMgr{sess: sess, picker: picker}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName})

	err := cmd.Execute()
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid project key")
}

func TestOpenCmd_WithPicker_InvalidKey_EmptySegmentPart(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{Name: testStoryName}}
	sess := &stubSess{}
	picker := &stubPickerClient{selectedKey: "github.com/seg1//seg3"} // empty middle segment
	mgr := &stubMgr{sess: sess, picker: picker}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName})

	err := cmd.Execute()
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid project key")
}

func TestOpenCmd_WithPicker_ExecArgvIsExeced(t *testing.T) {
	t.Parallel()

	const selectedKey = "github.com/kalbasit/swm"

	wantArgv := []string{"/usr/bin/tmux", "-S", "/tmp/feat-x.sock", "attach-session", "-t", "swm"}

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{
		Name: testStoryName,
		Projects: []coreStory.Project{
			{Host: testHost, Segments: []string{testOwner, testSegment}},
		},
	}}

	var gotArgv []string

	testExec := workspace.ExecFunc(func(_ string, argv []string, _ []string) error {
		gotArgv = argv

		return nil
	})

	sess := &stubSess{switchToExecArgv: wantArgv}
	picker := &stubPickerClient{selectedKey: selectedKey}
	mgr := &stubMgr{sess: sess, picker: picker}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop, workspace.WithExecFunc(testExec))
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute())
	require.Equal(t, wantArgv, gotArgv, "expected execFunc to be called with the argv from SwitchTo")
}

func TestOpenCmd_NoPicker_FallsBackToPhase1(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{
		Name: testStoryName,
		Projects: []coreStory.Project{
			{Host: testHost, Segments: []string{testOwner, testSegment}},
		},
	}}
	sess := &stubSess{}
	mgr := &stubMgr{sess: sess} // no picker configured
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute())

	// Phase 1 path: OpenWorkspace with all attached projects.
	require.NotNil(t, sess.lastOpenReq)
	require.Contains(t, sess.lastOpenReq.GetWorktreePaths(), "github.com/kalbasit/swm")
}

func TestOpenCmd_NoPicker_ExecArgvIsExeced(t *testing.T) {
	t.Parallel()

	wantArgv := []string{"/usr/bin/tmux", "attach-session", "-t", testStoryName}

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{
		Name: testStoryName,
		Projects: []coreStory.Project{
			{Host: testHost, Segments: []string{testOwner, testSegment}},
		},
	}}

	var gotArgv []string

	testExec := workspace.ExecFunc(func(_ string, argv []string, _ []string) error {
		gotArgv = argv

		return nil
	})

	sess := &stubSess{switchToExecArgv: wantArgv}
	mgr := &stubMgr{sess: sess} // no picker configured
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop, workspace.WithExecFunc(testExec))
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute())
	require.Equal(t, wantArgv, gotArgv, "expected execFunc to be called with the argv from SwitchTo")
}

// ─── story picker tests ───────────────────────────────────────────────────────

// TestOpenCmd_NoArgNoEnv_StoryPickerShown verifies that when no story is
// provided via arg or env, the store is listed (story picker runs) and the
// selected story's workspace is opened.
func TestOpenCmd_NoArgNoEnv_StoryPickerShown(t *testing.T) {
	t.Parallel()

	const selectedStory = testStoryName

	const selectedProject = "github.com/kalbasit/swm"

	feat := &coreStory.Story{
		Name: selectedStory,
		Projects: []coreStory.Project{
			{Host: testHost, Segments: []string{testOwner, testSegment}},
		},
	}
	dflt := &coreStory.Story{Name: testDefaultStory}

	store := &stubStore{
		listStories: []*coreStory.Story{feat, dflt},
		// getStory is returned for store.Get after story picker resolves the name.
		getStory: feat,
	}
	sess := &stubSess{}
	picker := &sequentialPickerClient{
		streams: []*stubPickStream{
			{selectedKey: selectedStory},   // story picker: select feat-x
			{selectedKey: selectedProject}, // project picker: select the project
		},
	}
	mgr := &stubMgr{sess: sess, picker: picker}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)
	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{}) // no positional arg

	require.NoError(t, cmd.Execute())

	// store.List was called by the story picker.
	require.True(t, store.listCalled, "expected List to be called by story picker")

	// Workspace was opened for the selected story.
	require.NotNil(t, sess.lastOpenReq)
	require.Equal(t, selectedStory, sess.lastOpenReq.GetStoryName())
}

// TestOpenCmd_PositionalArg_StoryPickerSkipped verifies that a positional arg
// bypasses the story picker entirely (store.List is not called).
func TestOpenCmd_PositionalArg_StoryPickerSkipped(t *testing.T) {
	t.Parallel()

	store := &stubStore{getStory: &coreStory.Story{Name: testStoryName}}
	sess := &stubSess{}
	// Picker is present — but should only be called for project selection, not story selection.
	picker := &stubPickerClient{cancelOnRecv: true} // abort project picker
	mgr := &stubMgr{sess: sess, picker: picker}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)
	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName}) // explicit arg

	require.NoError(t, cmd.Execute())

	// store.List must NOT have been called (story picker skipped).
	require.False(t, store.listCalled, "story picker must be skipped when positional arg is provided")
}

// TestOpenCmd_SWMStoryEnv_StoryPickerSkipped verifies that $SWM_STORY bypasses
// the story picker entirely (store.List is not called).
func TestOpenCmd_SWMStoryEnv_StoryPickerSkipped(t *testing.T) {
	t.Setenv("SWM_STORY", testStoryName)

	store := &stubStore{getStory: &coreStory.Story{Name: testStoryName}}
	sess := &stubSess{}
	picker := &stubPickerClient{cancelOnRecv: true}
	mgr := &stubMgr{sess: sess, picker: picker}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)
	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{})

	require.NoError(t, cmd.Execute())

	require.False(t, store.listCalled, "story picker must be skipped when $SWM_STORY is set")
}

// TestOpenCmd_StoryPickerAborted_ExitsClean verifies that cancelling the story
// picker exits with code 0 and does not open any workspace.
func TestOpenCmd_StoryPickerAborted_ExitsClean(t *testing.T) {
	t.Parallel()

	store := &stubStore{
		listStories: []*coreStory.Story{{Name: testStoryName}, {Name: testDefaultStory}},
		getStory:    &coreStory.Story{Name: testStoryName},
	}
	sess := &stubSess{}
	picker := &stubPickerClient{cancelOnRecv: true} // story picker aborted immediately
	mgr := &stubMgr{sess: sess, picker: picker}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)
	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{})

	require.NoError(t, cmd.Execute())
	require.Nil(t, sess.lastOpenReq, "no workspace should be opened when story picker is cancelled")
}

// TestOpenCmd_StoryPickerFailedPrecondition_FallsBackToDefault verifies that a
// FailedPrecondition from the story picker falls back to the default story.
func TestOpenCmd_StoryPickerFailedPrecondition_FallsBackToDefault(t *testing.T) {
	t.Parallel()

	defaultStory := &coreStory.Story{Name: testDefaultStory}

	store := &stubStore{
		listStories: []*coreStory.Story{defaultStory},
		getStory:    defaultStory,
	}
	sess := &stubSess{}
	// Story picker returns FailedPrecondition (no TTY).
	picker := &stubPickerClient{pickErr: status.Error(codes.FailedPrecondition, "no tty")}
	mgr := &stubMgr{sess: sess, picker: picker}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)
	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{})

	require.NoError(t, cmd.Execute())

	// Falls back to opening the default story.
	require.NotNil(t, sess.lastOpenReq, "workspace should be opened for default story")
	require.Equal(t, testDefaultStory, sess.lastOpenReq.GetStoryName())
}

// ─── kill-pane flag tests ─────────────────────────────────────────────────────

const (
	testOriginWorkspaceID = "/tmp/origin.sock"
	// testTargetWorkspaceID matches what stubSess.OpenWorkspace returns.
	testTargetWorkspaceID = "/tmp/feat-x.sock"
	// testTargetPaneGroupID matches what stubSess.OpenPaneGroup returns.
	testTargetPaneGroupID = "swm"
)

func TestOpenCmd_KillPane_TmuxPaneSet_PopulatesOriginFields(t *testing.T) {
	// Cannot be parallel — sets env vars.
	t.Setenv("TMUX_PANE", "%5")
	// Workspace ID is read from $TMUX on the host, not from CurrentContext.
	t.Setenv("TMUX", testOriginWorkspaceID+",12345,0")

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{
		Name: testStoryName,
		Projects: []coreStory.Project{
			{Host: testHost, Segments: []string{testOwner, testSegment}},
		},
	}}
	sess := &stubSess{
		// CurrentContext is consulted only for the safety guard (pane-group check).
		// Origin workspace ID now comes from $TMUX, not from this response.
		currentContextResp: &pluginv1.CurrentContextResponse{PaneGroupId: "some-other-pane-group"},
	}
	mgr := &stubMgr{sess: sess}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName, flagKillPane})

	require.NoError(t, cmd.Execute())
	require.NotNil(t, sess.lastSwitchReq)
	require.Equal(t, testOriginWorkspaceID, sess.lastSwitchReq.GetCloseOriginWorkspaceId())
	require.Equal(t, "%5", sess.lastSwitchReq.GetCloseOriginPaneId())
}

func TestOpenCmd_KillPane_TmuxPaneUnset_OmitsOriginFields(t *testing.T) {
	// Cannot be parallel — sets env vars.
	t.Setenv("TMUX_PANE", "")
	t.Setenv("TMUX", testOriginWorkspaceID+",12345,0")

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{
		Name: testStoryName,
		Projects: []coreStory.Project{
			{Host: testHost, Segments: []string{testOwner, testSegment}},
		},
	}}
	sess := &stubSess{}
	mgr := &stubMgr{sess: sess}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName, flagKillPane})

	require.NoError(t, cmd.Execute())
	require.NotNil(t, sess.lastSwitchReq)
	require.Empty(t, sess.lastSwitchReq.GetCloseOriginWorkspaceId(), "origin fields must be empty when TMUX_PANE is unset")
	require.Empty(t, sess.lastSwitchReq.GetCloseOriginPaneId())
}

func TestOpenCmd_KillPane_CurrentContextFails_StillPopulatesOriginFields(t *testing.T) {
	// Cannot be parallel — sets env vars.
	// Workspace ID now comes from $TMUX (not CurrentContext), so a CurrentContext
	// failure no longer prevents the origin fields from being set.
	t.Setenv("TMUX_PANE", "%5")
	t.Setenv("TMUX", testOriginWorkspaceID+",12345,0")

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{
		Name: testStoryName,
		Projects: []coreStory.Project{
			{Host: testHost, Segments: []string{testOwner, testSegment}},
		},
	}}
	sess := &stubSess{
		currentContextErr: errFakeHook,
	}
	mgr := &stubMgr{sess: sess}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName, flagKillPane})

	require.NoError(t, cmd.Execute())
	require.NotNil(t, sess.lastSwitchReq)
	require.Equal(t, testOriginWorkspaceID, sess.lastSwitchReq.GetCloseOriginWorkspaceId(),
		"origin workspace must be populated from $TMUX even when CurrentContext fails")
	require.Equal(t, "%5", sess.lastSwitchReq.GetCloseOriginPaneId())
}

func TestOpenCmd_KillPane_AlreadyInTarget_OmitsOriginFields(t *testing.T) {
	// Cannot be parallel — sets env vars.
	// When the caller is already inside the target workspace and pane group,
	// the switch is a no-op and we must not kill the current pane.
	t.Setenv("TMUX_PANE", "%5")
	t.Setenv("TMUX", testTargetWorkspaceID+",12345,0") // same socket as target workspace

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{
		Name: testStoryName,
		Projects: []coreStory.Project{
			{Host: testHost, Segments: []string{testOwner, testSegment}},
		},
	}}
	sess := &stubSess{
		// PaneGroupId matches what stubSess.OpenPaneGroup returns.
		currentContextResp: &pluginv1.CurrentContextResponse{PaneGroupId: testTargetPaneGroupID},
	}
	mgr := &stubMgr{sess: sess}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName, flagKillPane})

	require.NoError(t, cmd.Execute())
	require.NotNil(t, sess.lastSwitchReq)
	require.Empty(t, sess.lastSwitchReq.GetCloseOriginWorkspaceId(),
		"origin fields must be empty when already in the target workspace and pane group")
	require.Empty(t, sess.lastSwitchReq.GetCloseOriginPaneId())
}

func TestOpenCmd_KillPane_SameWorkspaceDifferentPaneGroup_PopulatesOriginFields(t *testing.T) {
	// Cannot be parallel — sets env vars.
	// Same workspace socket, different pane group — origin pane should be closed.
	t.Setenv("TMUX_PANE", "%5")
	t.Setenv("TMUX", testTargetWorkspaceID+",12345,0")

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{
		Name: testStoryName,
		Projects: []coreStory.Project{
			{Host: testHost, Segments: []string{testOwner, testSegment}},
		},
	}}
	sess := &stubSess{
		currentContextResp: &pluginv1.CurrentContextResponse{PaneGroupId: "other-project"},
	}
	mgr := &stubMgr{sess: sess}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName, flagKillPane})

	require.NoError(t, cmd.Execute())
	require.NotNil(t, sess.lastSwitchReq)
	require.Equal(t, testTargetWorkspaceID, sess.lastSwitchReq.GetCloseOriginWorkspaceId())
	require.Equal(t, "%5", sess.lastSwitchReq.GetCloseOriginPaneId())
}

func TestOpenCmd_KillPane_ZellijPaneSet_PopulatesOriginFields(t *testing.T) {
	// Cannot be parallel — sets env vars.
	// Clear tmux vars so Zellij detection is reached even when running inside tmux.
	t.Setenv("TMUX_PANE", "")
	t.Setenv("TMUX", "")
	t.Setenv("ZELLIJ_PANE_ID", "42")
	t.Setenv("ZELLIJ_SESSION_NAME", "/tmp/zellij-session")

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{
		Name: testStoryName,
		Projects: []coreStory.Project{
			{Host: testHost, Segments: []string{testOwner, testSegment}},
		},
	}}
	sess := &stubSess{
		currentContextResp: &pluginv1.CurrentContextResponse{PaneGroupId: "some-other-pane-group"},
	}
	mgr := &stubMgr{sess: sess}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName, flagKillPane})

	require.NoError(t, cmd.Execute())
	require.NotNil(t, sess.lastSwitchReq)
	require.Equal(t, "/tmp/zellij-session", sess.lastSwitchReq.GetCloseOriginWorkspaceId())
	require.Equal(t, "42", sess.lastSwitchReq.GetCloseOriginPaneId())
}

func TestOpenCmd_NoKillPane_OmitsOriginFields(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{
		Name: testStoryName,
		Projects: []coreStory.Project{
			{Host: testHost, Segments: []string{testOwner, testSegment}},
		},
	}}
	sess := &stubSess{}
	mgr := &stubMgr{sess: sess}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetArgs([]string{testStoryName}) // no --kill-pane

	require.NoError(t, cmd.Execute())
	require.NotNil(t, sess.lastSwitchReq)
	require.Empty(t, sess.lastSwitchReq.GetCloseOriginWorkspaceId(), "no origin fields without --kill-pane")
	require.Empty(t, sess.lastSwitchReq.GetCloseOriginPaneId())
}

// ─── stubs ───────────────────────────────────────────────────────────────────

// stubStore is a minimal story.Store.
type stubStore struct {
	getStory     *coreStory.Story
	getErr       error
	updateCalled bool
	listStories  []*coreStory.Story
	listErr      error
	listCalled   bool
}

func (s *stubStore) Create(context.Context, string, string) (*coreStory.Story, error) {
	panic("stub")
}

func (s *stubStore) Delete(context.Context, string) error { return nil }

func (s *stubStore) Get(_ context.Context, _ string) (*coreStory.Story, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}

	if s.getStory != nil {
		return s.getStory, nil
	}

	return &coreStory.Story{}, nil
}

func (s *stubStore) List(context.Context) ([]*coreStory.Story, error) {
	s.listCalled = true

	return s.listStories, s.listErr
}

func (s *stubStore) Update(_ context.Context, _ *coreStory.Story) error {
	s.updateCalled = true

	return nil
}

var _ coreStory.Store = (*stubStore)(nil)

// stubMgr implements pluginManager.
type stubMgr struct {
	sess   pluginv1.SessionClient
	vcs    pluginv1.VCSClient
	picker pluginv1.PickerClient
}

func (s *stubMgr) Get(_ context.Context, capability string) (any, error) {
	switch capability {
	case "session":
		if s.sess != nil {
			return s.sess, nil
		}
	case "vcs":
		if s.vcs != nil {
			return s.vcs, nil
		}
	case "picker":
		if s.picker != nil {
			return s.picker, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", errNoPlugin, capability)
}

// stubSess records session plugin calls.
type stubSess struct {
	lastOpenReq      *pluginv1.OpenWorkspaceRequest
	lastPaneGroupReq *pluginv1.OpenPaneGroupRequest
	lastSwitchReq    *pluginv1.SwitchToRequest
	switchToExecArgv []string // returned from SwitchTo when non-nil

	currentContextResp *pluginv1.CurrentContextResponse
	currentContextErr  error
}

func (s *stubSess) CloseWorkspace(
	context.Context,
	*pluginv1.CloseWorkspaceRequest,
	...grpc.CallOption,
) (*pluginv1.Empty, error) {
	panic("stub")
}

func (s *stubSess) CurrentContext(
	_ context.Context,
	_ *pluginv1.Empty,
	_ ...grpc.CallOption,
) (*pluginv1.CurrentContextResponse, error) {
	if s.currentContextErr != nil {
		return nil, s.currentContextErr
	}

	if s.currentContextResp != nil {
		return s.currentContextResp, nil
	}

	return &pluginv1.CurrentContextResponse{}, nil
}

func (s *stubSess) Info(
	context.Context,
	*pluginv1.Empty,
	...grpc.CallOption,
) (*pluginv1.SessionInfo, error) {
	panic("stub")
}

func (s *stubSess) IsInsideWorkspace(
	context.Context,
	*pluginv1.Empty,
	...grpc.CallOption,
) (*pluginv1.BoolValue, error) {
	panic("stub")
}

func (s *stubSess) ListWorkspaces(
	context.Context,
	*pluginv1.Empty,
	...grpc.CallOption,
) (grpc.ServerStreamingClient[pluginv1.Workspace], error) {
	return &eofStream{}, nil
}

func (s *stubSess) OpenPaneGroup(
	_ context.Context,
	req *pluginv1.OpenPaneGroupRequest,
	_ ...grpc.CallOption,
) (*pluginv1.PaneGroup, error) {
	s.lastPaneGroupReq = req

	return &pluginv1.PaneGroup{
		PaneGroupId: "swm",
		WorkspaceId: req.GetWorkspaceId(),
	}, nil
}

func (s *stubSess) OpenWorkspace(
	_ context.Context,
	req *pluginv1.OpenWorkspaceRequest,
	_ ...grpc.CallOption,
) (*pluginv1.Workspace, error) {
	s.lastOpenReq = req

	return &pluginv1.Workspace{
		WorkspaceId: "/tmp/feat-x.sock",
		StoryName:   req.GetStoryName(),
	}, nil
}

func (s *stubSess) SwitchTo(
	_ context.Context,
	req *pluginv1.SwitchToRequest,
	_ ...grpc.CallOption,
) (*pluginv1.SwitchToResponse, error) {
	s.lastSwitchReq = req

	return &pluginv1.SwitchToResponse{ExecArgv: s.switchToExecArgv}, nil
}

var _ pluginv1.SessionClient = (*stubSess)(nil)

// stubVCS records CreateWorktree calls.
type stubVCS struct {
	createCalled bool
}

func (v *stubVCS) Clone(context.Context, *pluginv1.CloneRequest, ...grpc.CallOption) (*pluginv1.CloneResponse, error) {
	panic("stub")
}

func (v *stubVCS) CreateWorktree(
	_ context.Context,
	_ *pluginv1.CreateWorktreeRequest,
	_ ...grpc.CallOption,
) (*pluginv1.Empty, error) {
	v.createCalled = true

	return &pluginv1.Empty{}, nil
}

func (v *stubVCS) DetectProjectAtPath(
	context.Context,
	*pluginv1.DetectAtPathRequest,
	...grpc.CallOption,
) (*pluginv1.ProjectID, error) {
	panic("stub")
}

func (v *stubVCS) Info(context.Context, *pluginv1.Empty, ...grpc.CallOption) (*pluginv1.VCSInfo, error) {
	panic("stub")
}

func (v *stubVCS) ListBranches(
	context.Context,
	*pluginv1.ListBranchesRequest,
	...grpc.CallOption,
) (grpc.ServerStreamingClient[pluginv1.Branch], error) {
	panic("stub")
}

func (v *stubVCS) ParseRemoteURL(
	context.Context,
	*pluginv1.ParseRemoteURLRequest,
	...grpc.CallOption,
) (*pluginv1.ProjectID, error) {
	panic("stub")
}

func (v *stubVCS) RemoveWorktree(
	context.Context,
	*pluginv1.RemoveWorktreeRequest,
	...grpc.CallOption,
) (*pluginv1.Empty, error) {
	panic("stub")
}

var _ pluginv1.VCSClient = (*stubVCS)(nil)

// stubPickerClient implements pluginv1.PickerClient.
type stubPickerClient struct {
	selectedKey  string
	cancelOnRecv bool
	pickErr      error
	recvErr      error // returned from stream.Recv() instead of a normal result
}

func (p *stubPickerClient) Info(
	context.Context,
	*pluginv1.Empty,
	...grpc.CallOption,
) (*pluginv1.PickerInfo, error) {
	return &pluginv1.PickerInfo{PluginInfo: &pluginv1.PluginInfo{Name: "stub"}}, nil
}

func (p *stubPickerClient) Pick(
	_ context.Context,
	_ ...grpc.CallOption,
) (grpc.BidiStreamingClient[pluginv1.PickItem, pluginv1.PickResult], error) {
	if p.pickErr != nil {
		return nil, p.pickErr
	}

	return &stubPickStream{selectedKey: p.selectedKey, cancel: p.cancelOnRecv, recvErr: p.recvErr}, nil
}

var _ pluginv1.PickerClient = (*stubPickerClient)(nil)

// stubPickStream implements grpc.BidiStreamingClient[PickItem, PickResult].
type stubPickStream struct {
	selectedKey string
	cancel      bool
	recvCalled  bool
	recvErr     error // returned from Recv() when set, before checking cancel/selectedKey
}

func (s *stubPickStream) CloseSend() error { return nil }

func (s *stubPickStream) Context() context.Context { return context.Background() }

func (s *stubPickStream) Header() (metadata.MD, error) { panic("stub") }

func (s *stubPickStream) Recv() (*pluginv1.PickResult, error) {
	if s.recvCalled {
		return nil, io.EOF
	}

	s.recvCalled = true

	if s.recvErr != nil {
		return nil, s.recvErr
	}

	if s.cancel {
		return nil, status.Error(codes.Aborted, "cancelled")
	}

	return &pluginv1.PickResult{Key: s.selectedKey}, nil
}

func (s *stubPickStream) RecvMsg(any) error { return nil }

func (s *stubPickStream) Send(*pluginv1.PickItem) error { return nil }

func (s *stubPickStream) SendMsg(any) error { return nil }

func (s *stubPickStream) Trailer() metadata.MD { return nil }

// sequentialPickerClient returns a new stubPickStream per Pick() call, in order.
// This lets tests drive both the story picker and the project picker independently.
type sequentialPickerClient struct {
	streams []*stubPickStream
	idx     int
}

func (s *sequentialPickerClient) Info(
	context.Context,
	*pluginv1.Empty,
	...grpc.CallOption,
) (*pluginv1.PickerInfo, error) {
	return &pluginv1.PickerInfo{PluginInfo: &pluginv1.PluginInfo{Name: "stub"}}, nil
}

func (s *sequentialPickerClient) Pick(
	_ context.Context,
	_ ...grpc.CallOption,
) (grpc.BidiStreamingClient[pluginv1.PickItem, pluginv1.PickResult], error) {
	if s.idx >= len(s.streams) {
		return &stubPickStream{cancel: true}, nil // extra calls cancel
	}

	stream := s.streams[s.idx]
	s.idx++

	return stream, nil
}

var _ pluginv1.PickerClient = (*sequentialPickerClient)(nil)

// eofStream is a server stream that immediately returns io.EOF.
type eofStream struct{}

func (e *eofStream) CloseSend() error                   { return nil }
func (e *eofStream) Context() context.Context           { return context.Background() }
func (e *eofStream) Header() (metadata.MD, error)       { panic("stub") }
func (e *eofStream) Recv() (*pluginv1.Workspace, error) { return nil, io.EOF }
func (e *eofStream) RecvMsg(any) error                  { panic("stub") }
func (e *eofStream) SendMsg(any) error                  { panic("stub") }
func (e *eofStream) Trailer() metadata.MD               { panic("stub") }
