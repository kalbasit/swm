package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"

	"github.com/kalbasit/swm/cmd/swm/internal/cli/workspace"
	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
)

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

	candidates := workspace.BuildCandidates(codeRoot, st, resolver)

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

	candidates := workspace.BuildCandidates(codeRoot, st, resolver)

	require.ElementsMatch(t, []string{
		"github.com/acme/app",
		"github.com/acme/infra",
	}, candidates)
}
