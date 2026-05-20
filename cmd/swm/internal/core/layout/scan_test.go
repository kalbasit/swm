package layout_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
)

func TestScanRepos_CorrectRepoSet(t *testing.T) {
	t.Parallel()

	codeRoot := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(codeRoot, "repositories/github.com/alice/app/.git"), 0o750))
	require.NoError(t, os.MkdirAll(filepath.Join(codeRoot, "repositories/github.com/alice/lib/.git"), 0o750))
	require.NoError(t, os.MkdirAll(filepath.Join(codeRoot, "repositories/gitlab.com/bob/tool/.git"), 0o750))

	resolver := layout.NewResolver(codeRoot, "_default")
	got, err := resolver.ScanRepos(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 3)

	keys := make(map[string]bool, len(got))
	for _, id := range got {
		keys[id.Host+"/"+strings.Join(id.Segments, "/")] = true
	}

	require.True(t, keys["github.com/alice/app"])
	require.True(t, keys["github.com/alice/lib"])
	require.True(t, keys["gitlab.com/bob/tool"])
}

func TestScanRepos_SubRepositoriesExcluded(t *testing.T) {
	t.Parallel()

	codeRoot := t.TempDir()

	repoDir := filepath.Join(codeRoot, "repositories/github.com/acme/infra")
	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, ".git"), 0o750))
	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, ".terraform/modules/vpc/.git"), 0o750))
	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, "tmp/clone/.git"), 0o750))

	resolver := layout.NewResolver(codeRoot, "_default")
	got, err := resolver.ScanRepos(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "github.com", got[0].Host)
	require.Equal(t, []string{"acme", "infra"}, got[0].Segments)
}

func TestScanRepos_HostDirNeverTreatedAsRepo(t *testing.T) {
	t.Parallel()

	codeRoot := t.TempDir()

	// .git directly inside a host dir — must NOT produce a ProjectID since hosts
	// require at least one segment.
	require.NoError(t, os.MkdirAll(filepath.Join(codeRoot, "repositories/github.com/.git"), 0o750))
	// Real repo at proper depth still returned.
	require.NoError(t, os.MkdirAll(filepath.Join(codeRoot, "repositories/github.com/user/repo/.git"), 0o750))

	resolver := layout.NewResolver(codeRoot, "_default")
	got, err := resolver.ScanRepos(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, []string{"user", "repo"}, got[0].Segments)
}

func TestScanRepos_EmptyRepositoriesDir(t *testing.T) {
	t.Parallel()

	codeRoot := t.TempDir()

	resolver := layout.NewResolver(codeRoot, "_default")
	got, err := resolver.ScanRepos(context.Background())
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestScanRepos_MissingRepositoriesDir(t *testing.T) {
	t.Parallel()

	resolver := layout.NewResolver("/nonexistent/code/root", "_default")
	got, err := resolver.ScanRepos(context.Background())
	require.NoError(t, err)
	require.Empty(t, got)
}
