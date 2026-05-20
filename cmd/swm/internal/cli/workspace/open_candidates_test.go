package workspace_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"
	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/cmd/swm/internal/cli/workspace"
	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
)

// stubLister is a test double for workspace.ProjectLister.
type stubLister struct{ projects []*pluginv1.ProjectID }

func (s *stubLister) Projects(_ context.Context) ([]*pluginv1.ProjectID, error) {
	return s.projects, nil
}

func TestBuildCandidates_SkipsSubRepositories(t *testing.T) {
	t.Parallel()

	codeRoot := t.TempDir()

	// Top-level project: github.com/acme/infra
	repoDir := filepath.Join(codeRoot, "repositories", "github.com", "acme", "infra")
	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, ".git"), 0o750))

	// Nested sub-repos inside the project (simulating tool caches and temp clones).
	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, ".terraform", "modules", "vpc", ".git"), 0o750))
	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, "tmp", "clone", ".git"), 0o750))

	resolver := layout.NewResolver(codeRoot, "_default")
	st := &coreStory.Story{}

	candidates := workspace.BuildCandidates(context.Background(), resolver, st)

	require.Equal(t, []string{"github.com/acme/infra"}, candidates)
}

func TestBuildCandidates_ReturnsSiblingRepos(t *testing.T) {
	t.Parallel()

	codeRoot := t.TempDir()

	// Two sibling top-level projects.
	require.NoError(t, os.MkdirAll(
		filepath.Join(codeRoot, "repositories", "github.com", "acme", "app", ".git"), 0o750,
	))
	require.NoError(t, os.MkdirAll(
		filepath.Join(codeRoot, "repositories", "github.com", "acme", "infra", ".git"), 0o750,
	))

	resolver := layout.NewResolver(codeRoot, "_default")
	st := &coreStory.Story{}

	candidates := workspace.BuildCandidates(context.Background(), resolver, st)

	require.ElementsMatch(t, []string{
		"github.com/acme/app",
		"github.com/acme/infra",
	}, candidates)
}

func TestBuildCandidates_AttachedProjectsFirst(t *testing.T) {
	t.Parallel()

	lister := &stubLister{projects: []*pluginv1.ProjectID{
		{Host: testHost, Segments: []string{testOwner, "other"}},
	}}

	st := &coreStory.Story{
		Projects: []coreStory.Project{
			{Host: testHost, Segments: []string{testOwner, "attached"}},
		},
	}

	candidates := workspace.BuildCandidates(context.Background(), lister, st)

	require.Equal(t, testHost+"/"+testOwner+"/attached", candidates[0], "attached project must be first")
	require.Contains(t, candidates, testHost+"/"+testOwner+"/other")
}

func TestBuildCandidates_DeduplicatesAttachedAndOnDisk(t *testing.T) {
	t.Parallel()

	lister := &stubLister{projects: []*pluginv1.ProjectID{
		{Host: testHost, Segments: []string{testOwner, "repo"}},
	}}

	st := &coreStory.Story{
		Projects: []coreStory.Project{
			{Host: testHost, Segments: []string{testOwner, "repo"}},
		},
	}

	candidates := workspace.BuildCandidates(context.Background(), lister, st)

	require.Equal(t, []string{testHost + "/" + testOwner + "/repo"}, candidates, "duplicate must be removed")
}
