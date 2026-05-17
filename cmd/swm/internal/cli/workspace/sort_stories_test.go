package workspace_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"

	"github.com/kalbasit/swm/cmd/swm/internal/cli/workspace"
)

const (
	testSortStoryOld   = "old"
	testSortStoryNew   = "new"
	testSortStoryAlpha = "alpha"
	testSortStoryBeta  = "beta"
)

var sortBase = time.Date(2026, 5, 17, 0, 0, 0, 0, time.UTC)

func TestSortStoriesForPicker_DescendingByCreatedAt(t *testing.T) {
	t.Parallel()

	stories := []*coreStory.Story{
		{Name: testSortStoryOld, CreatedAt: sortBase.Add(-7 * 24 * time.Hour)},
		{Name: testSortStoryNew, CreatedAt: sortBase.Add(-1 * time.Hour)},
		{Name: "mid", CreatedAt: sortBase.Add(-3 * 24 * time.Hour)},
	}

	got := workspace.SortStoriesForPicker(stories)

	require.Equal(t, []string{testSortStoryNew, "mid", testSortStoryOld}, names(got))
}

func TestSortStoriesForPicker_DefaultPinnedLast(t *testing.T) {
	t.Parallel()

	// _default has a very recent CreatedAt — it must still appear last.
	stories := []*coreStory.Story{
		{Name: testDefaultStory, CreatedAt: sortBase},
		{Name: testSortStoryOld, CreatedAt: sortBase.Add(-7 * 24 * time.Hour)},
		{Name: testSortStoryNew, CreatedAt: sortBase.Add(-1 * time.Hour)},
	}

	got := workspace.SortStoriesForPicker(stories)

	require.Equal(t, testSortStoryNew, got[0].Name)
	require.Equal(t, testSortStoryOld, got[1].Name)
	require.Equal(t, testDefaultStory, got[2].Name)
}

func TestSortStoriesForPicker_TiesOrderedLexicographically(t *testing.T) {
	t.Parallel()

	same := sortBase.Add(-2 * time.Hour)

	stories := []*coreStory.Story{
		{Name: "zeta", CreatedAt: same},
		{Name: testSortStoryAlpha, CreatedAt: same},
		{Name: testSortStoryBeta, CreatedAt: same},
	}

	got := workspace.SortStoriesForPicker(stories)

	require.Equal(t, []string{testSortStoryAlpha, testSortStoryBeta, "zeta"}, names(got))
}

func TestSortStoriesForPicker_OnlyDefault(t *testing.T) {
	t.Parallel()

	stories := []*coreStory.Story{
		{Name: testDefaultStory, CreatedAt: sortBase},
	}

	got := workspace.SortStoriesForPicker(stories)

	require.Len(t, got, 1)
	require.Equal(t, testDefaultStory, got[0].Name)
}

func TestSortStoriesForPicker_DoesNotMutateInput(t *testing.T) {
	t.Parallel()

	stories := []*coreStory.Story{
		{Name: "b", CreatedAt: sortBase.Add(-2 * time.Hour)},
		{Name: "a", CreatedAt: sortBase.Add(-1 * time.Hour)},
	}

	origOrder := []*coreStory.Story{stories[0], stories[1]}

	workspace.SortStoriesForPicker(stories)

	// Original slice should be unchanged.
	require.Equal(t, origOrder[0].Name, stories[0].Name)
	require.Equal(t, origOrder[1].Name, stories[1].Name)
}

// names extracts story names for compact assertion.
func names(ss []*coreStory.Story) []string {
	out := make([]string, len(ss))

	for i, s := range ss {
		out[i] = s.Name
	}

	return out
}
