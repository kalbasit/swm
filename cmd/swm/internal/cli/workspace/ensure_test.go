package workspace_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"
	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/cmd/swm/internal/cli/workspace"
	"github.com/kalbasit/swm/cmd/swm/internal/config"
	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
	"github.com/kalbasit/swm/cmd/swm/internal/hookexec"
)

// testWorkspaceID is what stubSess.OpenWorkspace returns.
const testWorkspaceID = "/tmp/feat-x.sock"

var errHookFailed = errors.New("hook failed")

// recordingSess records every pane group opened, not just the last one. stubSess
// keeps only the most recent call, which is enough for a command that opens one
// group and not for a command whose whole point is opening several.
type recordingSess struct {
	stubSess

	paneGroupReqs   []*pluginv1.OpenPaneGroupRequest
	openWorkspaceN  int
	openWorkspaceEr error
}

func (s *recordingSess) OpenPaneGroup(
	ctx context.Context,
	req *pluginv1.OpenPaneGroupRequest,
	opts ...grpc.CallOption,
) (*pluginv1.PaneGroup, error) {
	s.paneGroupReqs = append(s.paneGroupReqs, req)

	return s.stubSess.OpenPaneGroup(ctx, req, opts...)
}

func (s *recordingSess) OpenWorkspace(
	ctx context.Context,
	req *pluginv1.OpenWorkspaceRequest,
	opts ...grpc.CallOption,
) (*pluginv1.Workspace, error) {
	s.openWorkspaceN++

	if s.openWorkspaceEr != nil {
		return nil, s.openWorkspaceEr
	}

	return s.stubSess.OpenWorkspace(ctx, req, opts...)
}

// projects builds n distinct attached projects.
func projects(names ...string) []coreStory.Project {
	ps := make([]coreStory.Project, 0, len(names))
	for _, n := range names {
		ps = append(ps, coreStory.Project{Host: testHost, Segments: []string{testOwner, n}})
	}

	return ps
}

// ensureCmd builds the command with stdout and stderr kept apart.
//
// Note what these tests can and cannot see: cobra's Print/Println write to
// OutOrStderr(), which returns the *out* writer once SetOut has been called. So
// no test here can distinguish printing to stdout from printing to stderr -- both
// land in the same buffer. That the id reaches stdout, which is the only thing
// that makes `ws=$(swm workspace ensure x)` work, is verified by running the
// built binary; a run of it is what found the id going to stderr in the first
// place. These tests assert the content, not the stream.
func ensureCmd(
	t *testing.T,
	store coreStory.Store,
	sess pluginv1.SessionClient,
	hooks hookexec.Runner,
	args ...string,
) (*cobra.Command, *bytes.Buffer) {
	t.Helper()

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	mgr := &stubMgr{sess: sess}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	cmd := workspace.NewEnsureCmd(cfg, store, mgr, resolver, hooks)

	var stdout, stderr bytes.Buffer

	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)

	return cmd, &stdout
}

func TestEnsureCmd_OpensWorkspaceAndEveryGroupAndPrintsID(t *testing.T) {
	t.Parallel()

	store := &stubStore{getStory: &coreStory.Story{
		Name:     testStoryName,
		Projects: projects("swm", "steward"),
	}}
	sess := &recordingSess{}

	cmd, out := ensureCmd(t, store, sess, hookexec.Noop, testStoryName)
	require.NoError(t, cmd.Execute())

	require.Len(t, sess.paneGroupReqs, 2, "expected one pane group per attached project")
	require.Equal(t, testWorkspaceID+"\n", out.String(), "stdout must be the workspace id and nothing else")

	// Every attached project's worktree path reaches OpenWorkspace.
	require.Len(t, sess.lastOpenReq.GetWorktreePaths(), 2)
	require.Contains(t, sess.lastOpenReq.GetWorktreePaths(), testHost+"/"+testOwner+"/swm")
	require.Contains(t, sess.lastOpenReq.GetWorktreePaths(), testHost+"/"+testOwner+"/steward")
}

func TestEnsureCmd_NeverSwitchesAndNeverExecs(t *testing.T) {
	t.Parallel()

	store := &stubStore{getStory: &coreStory.Story{
		Name:     testStoryName,
		Projects: projects("swm"),
	}}
	// A SwitchTo response carrying an argv is what makes `open` replace its
	// process. ensure must not even ask.
	sess := &recordingSess{stubSess: stubSess{switchToExecArgv: []string{"tmux", "attach"}}}

	cmd, _ := ensureCmd(t, store, sess, hookexec.Noop, testStoryName)
	require.NoError(t, cmd.Execute())

	require.Nil(t, sess.lastSwitchReq, "ensure must never call SwitchTo")
}

func TestEnsureCmd_IsIdempotent(t *testing.T) {
	t.Parallel()

	story := &coreStory.Story{Name: testStoryName, Projects: projects("swm")}

	first := &recordingSess{}
	cmd1, out1 := ensureCmd(t, &stubStore{getStory: story}, first, hookexec.Noop, testStoryName)
	require.NoError(t, cmd1.Execute())

	second := &recordingSess{}
	cmd2, out2 := ensureCmd(t, &stubStore{getStory: story}, second, hookexec.Noop, testStoryName)
	require.NoError(t, cmd2.Execute())

	require.Equal(t, out1.String(), out2.String(), "the same story must report the same workspace id")
	require.Len(t, second.paneGroupReqs, len(first.paneGroupReqs))
}

func TestEnsureCmd_StoryWithNoProjectsStillYieldsAWorkspace(t *testing.T) {
	t.Parallel()

	store := &stubStore{getStory: &coreStory.Story{Name: testStoryName}}
	sess := &recordingSess{}

	cmd, out := ensureCmd(t, store, sess, hookexec.Noop, testStoryName)
	require.NoError(t, cmd.Execute())

	require.Equal(t, 1, sess.openWorkspaceN)
	require.Empty(t, sess.paneGroupReqs, "no projects means no pane groups")
	require.Equal(t, testWorkspaceID+"\n", out.String())
}

func TestEnsureCmd_UnknownStoryIsRefusedAndNotCreated(t *testing.T) {
	t.Parallel()

	store := &stubStore{getErr: coreStory.ErrStoryNotFound}
	sess := &recordingSess{}

	cmd, _ := ensureCmd(t, store, sess, hookexec.Noop, "no-such-story")
	err := cmd.Execute()

	require.Error(t, err)
	require.ErrorIs(t, err, coreStory.ErrStoryNotFound)
	require.Contains(t, err.Error(), "no-such-story", "the error must name the story")
	require.False(t, store.createCalled, "ensure must never create a story")
	require.Zero(t, sess.openWorkspaceN, "nothing is opened for a story that does not exist")
}

func TestEnsureCmd_NoPickerIsConsulted(t *testing.T) {
	// Not parallel: t.Setenv cannot be used in a parallel test, and this case
	// is specifically about story resolution falling through an unset
	// $SWM_STORY to the configured default.
	t.Setenv("SWM_STORY", "")

	store := &stubStore{getStory: &coreStory.Story{Name: testDefaultStory}}
	sess := &recordingSess{}
	picker := &stubPickerClient{selectedKey: "should-not-be-used"}

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	mgr := &stubMgr{sess: sess, picker: picker}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	cmd := workspace.NewEnsureCmd(cfg, store, mgr, resolver, hookexec.Noop)

	var stdout, stderr bytes.Buffer

	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(nil)

	require.NoError(t, cmd.Execute())
	require.Equal(t, testDefaultStory, sess.lastOpenReq.GetStoryName(), "the default story is used, not a picked one")
}

func TestEnsureCmd_FailingPreHookOpensNothing(t *testing.T) {
	t.Parallel()

	store := &stubStore{getStory: &coreStory.Story{
		Name:     testStoryName,
		Projects: projects("swm"),
	}}
	sess := &recordingSess{}

	hooks := hookexec.RunnerFunc(func(_ context.Context, cfg hookexec.RunConfig) error {
		if cfg.Event == "pre-workspace-open" {
			return errHookFailed
		}

		return nil
	})

	cmd, _ := ensureCmd(t, store, sess, hooks, testStoryName)
	err := cmd.Execute()

	require.Error(t, err)
	require.ErrorIs(t, err, errHookFailed)
	require.Zero(t, sess.openWorkspaceN, "a failing pre-hook must abort before anything is opened")
	require.Empty(t, sess.paneGroupReqs)
}

func TestEnsureCmd_FailingPostHookStillSucceeds(t *testing.T) {
	t.Parallel()

	store := &stubStore{getStory: &coreStory.Story{
		Name:     testStoryName,
		Projects: projects("swm"),
	}}
	sess := &recordingSess{}

	hooks := hookexec.RunnerFunc(func(_ context.Context, cfg hookexec.RunConfig) error {
		if cfg.Event == "post-workspace-open" {
			return errHookFailed
		}

		return nil
	})

	cmd, out := ensureCmd(t, store, sess, hooks, testStoryName)

	require.NoError(t, cmd.Execute(), "a post hook failure must not fail the command")
	require.Equal(t, testWorkspaceID+"\n", out.String())
}

func TestEnsureCmd_JSONOutput(t *testing.T) {
	t.Parallel()

	store := &stubStore{getStory: &coreStory.Story{
		Name:     testStoryName,
		Projects: projects("swm", "steward"),
	}}
	sess := &recordingSess{}

	cmd, out := ensureCmd(t, store, sess, hookexec.Noop, testStoryName, "--json")
	require.NoError(t, cmd.Execute())

	var got struct {
		WorkspaceID string `json:"workspace_id"`
		PaneGroups  []struct {
			Project     string `json:"project"`
			PaneGroupID string `json:"pane_group_id"`
		} `json:"pane_groups"`
	}

	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	require.Equal(t, testWorkspaceID, got.WorkspaceID)
	require.Len(t, got.PaneGroups, 2)
	require.Equal(t, testHost+"/"+testOwner+"/swm", got.PaneGroups[0].Project)
}

func TestEnsureCmd_OpenWorkspaceError(t *testing.T) {
	t.Parallel()

	store := &stubStore{getStory: &coreStory.Story{
		Name:     testStoryName,
		Projects: projects("swm"),
	}}
	sess := &recordingSess{openWorkspaceEr: errHookFailed}

	cmd, _ := ensureCmd(t, store, sess, hookexec.Noop, testStoryName)
	err := cmd.Execute()

	require.Error(t, err)
	require.ErrorIs(t, err, errHookFailed)
}

// TestOpenCmd_OpensOnlyTheFirstProjectsGroup guards the boundary the extraction
// created. `ensure` opens a group per project; `open` must keep opening exactly
// one, because the proposal says `open` is unchanged and three tmux windows
// where there was one is a change.
func TestOpenCmd_OpensOnlyTheFirstProjectsGroup(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{CodeRoot: testCodeRoot, DefaultStory: testDefaultStory}
	store := &stubStore{getStory: &coreStory.Story{
		Name:     testStoryName,
		Projects: projects("swm", "steward", "ncps"),
	}}
	sess := &recordingSess{}
	mgr := &stubMgr{sess: sess}
	resolver := layout.NewResolver(testCodeRoot, testDefaultStory)

	cmd := workspace.NewOpenCmd(cfg, store, mgr, resolver, hookexec.Noop)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute())

	require.Len(t, sess.paneGroupReqs, 1, "open must still open exactly one pane group")
	require.True(t,
		strings.HasSuffix(sess.paneGroupReqs[0].GetProjectId().GetSegments()[1], "swm"),
		"and it must be the first attached project's")
}

func TestEnsureCmd_CompletesStoryNames(t *testing.T) {
	t.Parallel()

	store := &stubStore{listStories: []*coreStory.Story{
		{Name: testStoryName},
		{Name: testStoryName + "-two"},
	}}

	cmd, _ := ensureCmd(t, store, &recordingSess{}, hookexec.Noop)

	names, directive := cmd.ValidArgsFunction(cmd, nil, "")
	require.Equal(t, []string{testStoryName, testStoryName + "-two"}, names)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive,
		"filenames must never be offered where a story name belongs")
}

func TestEnsureCmd_CompletesNothingOnceTheArgumentIsGiven(t *testing.T) {
	t.Parallel()

	store := &stubStore{listStories: []*coreStory.Story{{Name: testStoryName}}}
	cmd, _ := ensureCmd(t, store, &recordingSess{}, hookexec.Noop)

	names, directive := cmd.ValidArgsFunction(cmd, []string{testStoryName}, "")
	require.Empty(t, names)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}

func TestEnsureCmd_CompletionDegradesOnStoreError(t *testing.T) {
	t.Parallel()

	store := &stubStore{listErr: errHookFailed}
	cmd, _ := ensureCmd(t, store, &recordingSess{}, hookexec.Noop)

	names, directive := cmd.ValidArgsFunction(cmd, nil, "")
	require.Empty(t, names, "a store error offers no candidates rather than filenames")
	require.Equal(t, cobra.ShellCompDirectiveError, directive)
}
