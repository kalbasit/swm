package workspace_test

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/require"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"

	"github.com/kalbasit/swm/cmd/swm/internal/cli/workspace"
)

var displayNow = time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)

func makeStory(name, branch string, createdAt time.Time, projects ...string) *coreStory.Story {
	ps := make([]coreStory.Project, 0, len(projects))

	for _, p := range projects {
		parts := strings.SplitN(p, "/", 2)
		segs := strings.Split(parts[1], "/")
		ps = append(ps, coreStory.Project{Host: parts[0], Segments: segs})
	}

	return &coreStory.Story{
		Name:       name,
		BranchName: branch,
		CreatedAt:  createdAt,
		Projects:   ps,
	}
}

func TestBuildStoryDisplay_BranchOmittedWhenSameAsName(t *testing.T) {
	t.Parallel()

	s := makeStory("feat/my-feature", "feat/my-feature", displayNow.Add(-3*24*time.Hour))
	display := workspace.BuildStoryDisplay(s, 200, displayNow)

	require.Contains(t, display, "feat/my-feature")
	require.NotContains(t, display, "(feat/my-feature)")
}

func TestBuildStoryDisplay_BranchShownWhenDifferent(t *testing.T) {
	t.Parallel()

	s := makeStory("jira-42", "fix/JIRA-42-crash", displayNow.Add(-2*time.Hour))
	display := workspace.BuildStoryDisplay(s, 200, displayNow)

	require.Contains(t, display, "jira-42")
	require.Contains(t, display, "(fix/JIRA-42-crash)")
}

func TestBuildStoryDisplay_DefaultStoreyLabel(t *testing.T) {
	t.Parallel()

	s := makeStory("_default", "main", displayNow.Add(-2*365*24*time.Hour))
	display := workspace.BuildStoryDisplay(s, 200, displayNow)

	require.True(t, strings.HasPrefix(display, "_default (main repo)"),
		"display should start with _default (main repo), got: %q", display)
}

func TestBuildStoryDisplay_DefaultStoreyLabel_EvenWhenBranchSame(t *testing.T) {
	t.Parallel()

	// _default with branch == name: still shows (main repo)
	s := makeStory("_default", "_default", displayNow.Add(-2*365*24*time.Hour))
	display := workspace.BuildStoryDisplay(s, 200, displayNow)

	require.Contains(t, display, "_default (main repo)")
}

func TestBuildStoryDisplay_ProjectsJoinedWithDot(t *testing.T) {
	t.Parallel()

	s := makeStory("feat/x", "feat/x", displayNow.Add(-24*time.Hour),
		"github.com/kalbasit/swm", "github.com/kalbasit/dotfiles")
	display := workspace.BuildStoryDisplay(s, 200, displayNow)

	require.Contains(t, display, "github.com/kalbasit/swm · github.com/kalbasit/dotfiles")
}

func TestBuildStoryDisplay_ZeroProjects_NoProjectSuffix(t *testing.T) {
	t.Parallel()

	s := makeStory("feat/x", "feat/x", displayNow.Add(-24*time.Hour))
	display := workspace.BuildStoryDisplay(s, 200, displayNow)

	require.NotContains(t, display, "·")
}

func TestBuildStoryDisplay_DoesNotExceedWidth(t *testing.T) {
	t.Parallel()

	s := makeStory(
		"feat/my-feature", "feat/WM-123-long-branch-name", displayNow.Add(-3*24*time.Hour),
		"github.com/kalbasit/swm",
		"github.com/kalbasit/dotfiles",
		"github.com/kalbasit/home",
		"github.com/org/very-long-repo-name",
	)
	display := workspace.BuildStoryDisplay(s, 80, displayNow)

	require.LessOrEqual(t, utf8.RuneCountInString(display), 80, "display %q exceeds 80 cols", display)
}

func TestBuildStoryDisplay_ProjectsTrimmedWithEllipsis(t *testing.T) {
	t.Parallel()

	s := makeStory(
		"feat/x", "feat/x", displayNow.Add(-24*time.Hour),
		"github.com/kalbasit/swm",
		"github.com/kalbasit/dotfiles",
		"github.com/kalbasit/home",
		"github.com/org/very-long-repo-name",
	)
	display := workspace.BuildStoryDisplay(s, 60, displayNow)

	require.LessOrEqual(t, utf8.RuneCountInString(display), 60)
	// Some projects are trimmed — ellipsis or fewer projects shown.
	require.NotContains(t, display, "github.com/org/very-long-repo-name",
		"overflow project should be trimmed")
}

func TestBuildStoryDisplay_ProjectsTrimmedShowsMaxFittingProjects(t *testing.T) {
	t.Parallel()

	// 3 projects, each 15 runes. Width 55 fits 2+ellipsis (52 runes) but not all 3 (66 runes).
	// The trimmer must show 2 projects, not fall back to 1.
	s := makeStory("x", "x", displayNow.Add(-24*time.Hour),
		"github.com/a/p1",
		"github.com/a/p2",
		"github.com/a/p3",
	)
	display := workspace.BuildStoryDisplay(s, 55, displayNow)

	require.LessOrEqual(t, utf8.RuneCountInString(display), 55)
	require.Contains(t, display, "github.com/a/p2",
		"second project must be visible when 2 projects + ellipsis fit within width")
}

func TestBuildStoryDisplay_StoryNamePreservedAtMinimum(t *testing.T) {
	t.Parallel()

	// Even at a very narrow width the story name must survive.
	s := makeStory("feat/x", "feat/x", displayNow.Add(-24*time.Hour))
	display := workspace.BuildStoryDisplay(s, 10, displayNow)

	require.Contains(t, display, "feat/x")
}

func TestBuildStoryDisplay_BranchTruncatedWhenNecessary(t *testing.T) {
	t.Parallel()

	s := makeStory(
		"feat/x", "feat/WM-999-a-very-long-branch-name-that-does-not-fit", displayNow.Add(-24*time.Hour),
		"github.com/kalbasit/swm",
	)
	// Wide enough for name and age but not the full branch.
	display := workspace.BuildStoryDisplay(s, 50, displayNow)

	require.LessOrEqual(t, utf8.RuneCountInString(display), 50)
	require.Contains(t, display, "feat/x") // story name always present
}

func TestBuildStoryDisplay_ContainsAge(t *testing.T) {
	t.Parallel()

	s := makeStory("feat/x", "feat/x", displayNow.Add(-3*24*time.Hour))
	display := workspace.BuildStoryDisplay(s, 200, displayNow)

	require.Contains(t, display, "3d ago")
}
