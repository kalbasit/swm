package story_test

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"

	"github.com/kalbasit/swm/cmd/swm/internal/cli/story"
	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
)

const (
	showCodeRoot     = "/code"
	showDefaultStory = "_default"
)

type shownStory struct {
	Name       string `json:"name"`
	BranchName string `json:"branch_name"`
	Projects   []struct {
		Key          string `json:"key"`
		WorktreePath string `json:"worktree_path"`
	} `json:"projects"`
}

func storyWithProjects(name string, repos ...string) *coreStory.Story {
	st := &coreStory.Story{
		Name:       name,
		BranchName: "user/wnasreddine/" + name,
		CreatedAt:  time.Unix(0, 0).UTC(),
	}

	for _, r := range repos {
		st.Projects = append(st.Projects, coreStory.Project{
			Host:     testGitHubHost,
			Segments: []string{testKalbasitOrg, r},
		})
	}

	return st
}

func showCmd(t *testing.T, store coreStory.Store, args ...string) (*cobra.Command, *bytes.Buffer) {
	t.Helper()

	resolver := layout.NewResolver(showCodeRoot, showDefaultStory)
	cmd := story.NewShowCmd(store, resolver, showDefaultStory)

	var stdout, stderr bytes.Buffer

	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)

	return cmd, &stdout
}

func decodeShown(t *testing.T, out *bytes.Buffer) shownStory {
	t.Helper()

	var got shownStory

	require.NoError(t, json.Unmarshal(out.Bytes(), &got))

	return got
}

// TestShowReportsBranchNameAndWorktreePaths is the whole point: both facts
// existed only inside the process, which left a caller reconstructing them.
func TestShowReportsBranchNameAndWorktreePaths(t *testing.T) {
	t.Parallel()

	store := &stubStore{getStory: storyWithProjects(testStoryName, testSWMRepo, "steward")}

	cmd, out := showCmd(t, store, testStoryName, "--json")
	require.NoError(t, cmd.Execute())

	got := decodeShown(t, out)
	require.Equal(t, "user/wnasreddine/"+testStoryName, got.BranchName)
	require.Len(t, got.Projects, 2)

	byKey := map[string]string{}
	for _, p := range got.Projects {
		byKey[p.Key] = p.WorktreePath
	}

	require.Equal(t,
		"/code/stories/"+testStoryName+"/github.com/kalbasit/swm",
		byKey["github.com/kalbasit/swm"])
	require.Equal(t,
		"/code/stories/"+testStoryName+"/github.com/kalbasit/steward",
		byKey["github.com/kalbasit/steward"])
}

// TestShowUsesTheResolverForTheDefaultStory guards the one case a command
// reimplementing the layout would get wrong: the default story has no separate
// worktree, so its projects resolve to their canonical clones.
func TestShowUsesTheResolverForTheDefaultStory(t *testing.T) {
	t.Parallel()

	store := &stubStore{getStory: storyWithProjects(showDefaultStory, testSWMRepo)}

	cmd, out := showCmd(t, store, showDefaultStory, "--json")
	require.NoError(t, cmd.Execute())

	got := decodeShown(t, out)
	require.Len(t, got.Projects, 1)
	require.Equal(t,
		"/code/repositories/github.com/kalbasit/swm",
		got.Projects[0].WorktreePath,
		"the default story resolves to the canonical clone, not a worktree")
}

func TestShowResolvesFromEnvThenDefault(t *testing.T) {
	// Not parallel: t.Setenv.
	t.Setenv("SWM_STORY", testBugName)

	store := &stubStore{getStory: storyWithProjects(testBugName)}

	cmd, _ := showCmd(t, store, "--json")
	require.NoError(t, cmd.Execute())
	require.Equal(t, testBugName, store.lastGetName, "$SWM_STORY resolves the story")

	t.Setenv("SWM_STORY", "")

	fallback := &stubStore{getStory: storyWithProjects(showDefaultStory)}

	cmd, _ = showCmd(t, fallback, "--json")
	require.NoError(t, cmd.Execute())
	require.Equal(t, showDefaultStory, fallback.lastGetName, "with nothing set, the default story is used")
}

func TestShowArgumentBeatsEnv(t *testing.T) {
	// Not parallel: t.Setenv.
	t.Setenv("SWM_STORY", testBugName)

	store := &stubStore{getStory: storyWithProjects(testStoryName)}

	cmd, _ := showCmd(t, store, testStoryName, "--json")
	require.NoError(t, cmd.Execute())
	require.Equal(t, testStoryName, store.lastGetName)
}

func TestShowRefusesAnUnknownStory(t *testing.T) {
	t.Parallel()

	store := &stubStore{getErr: errNotFound}

	cmd, out := showCmd(t, store, "no-such-story", "--json")
	err := cmd.Execute()

	require.Error(t, err)
	require.Contains(t, err.Error(), "no-such-story", "the error must name the story")

	// Not asserting empty output: this constructs the subcommand standalone,
	// where cobra prints usage. The root command sets SilenceUsage, so that
	// never happens in use. What must not appear is a record.
	require.NotContains(t, out.String(), `"branch_name"`,
		"no story record is printed for a story that does not exist")
}

// TestShowReportsAStoryWithNoProjects: a story nothing is attached to is a
// story, not an error.
func TestShowReportsAStoryWithNoProjects(t *testing.T) {
	t.Parallel()

	store := &stubStore{getStory: storyWithProjects(testStoryName)}

	cmd, out := showCmd(t, store, testStoryName, "--json")
	require.NoError(t, cmd.Execute())

	got := decodeShown(t, out)
	require.Equal(t, testStoryName, got.Name)
	require.Empty(t, got.Projects)
}

// TestShowNeedsNoMultiplexer is what makes this usable from a daemon, and what
// lets it answer for a story whose workspace is closed.
func TestShowNeedsNoMultiplexer(t *testing.T) {
	t.Parallel()

	store := &stubStore{getStory: storyWithProjects(testStoryName, testSWMRepo)}

	// No plugin manager is passed to NewShowCmd at all, so no session plugin
	// can be loaded. Constructing the command is the assertion.
	cmd, out := showCmd(t, store, testStoryName, "--json")
	require.NoError(t, cmd.Execute())
	require.NotEmpty(t, out.String())
}

func TestShowHumanReadableOutput(t *testing.T) {
	t.Parallel()

	store := &stubStore{getStory: storyWithProjects(testStoryName, testSWMRepo)}

	cmd, out := showCmd(t, store, testStoryName)
	require.NoError(t, cmd.Execute())

	text := out.String()
	require.Contains(t, text, testStoryName)
	require.Contains(t, text, "user/wnasreddine/"+testStoryName)
	require.Contains(t, text, "/code/stories/"+testStoryName+"/github.com/kalbasit/swm")
	require.Contains(t, text, "github.com/kalbasit/swm")
}

func TestShowCompletesStoryNames(t *testing.T) {
	t.Parallel()

	store := &stubStore{listStories: []*coreStory.Story{{Name: testStoryName}, {Name: testBugName}}}

	cmd, _ := showCmd(t, store)

	names, directive := cmd.ValidArgsFunction(cmd, nil, "")
	require.Equal(t, []string{testStoryName, testBugName}, names)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
}

func TestShowCompletionDegradesOnStoreError(t *testing.T) {
	t.Parallel()

	store := &stubStore{listErr: errFakeStore}

	cmd, _ := showCmd(t, store)

	names, directive := cmd.ValidArgsFunction(cmd, nil, "")
	require.Empty(t, names)
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive,
		"not Error: cobra's bash script returns on Error before disabling default completion")
}
