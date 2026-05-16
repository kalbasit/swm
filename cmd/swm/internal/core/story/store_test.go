package story_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kalbasit/swm/cmd/swm/internal/core/story"
)

const (
	testHost    = "github.com"
	testOwner   = "kalbasit"
	testProject = "swm"
)

func newTestStore(t *testing.T) story.Store {
	t.Helper()

	dir := t.TempDir()

	return story.NewJSONStore(filepath.Join(dir, "stories"))
}

func TestCreate_PersistsJSON(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	before := time.Now().Truncate(time.Second)

	s, err := store.Create(context.Background(), "feat-x", "feat/feat-x")
	require.NoError(t, err)
	require.Equal(t, "feat-x", s.Name)
	require.Equal(t, "feat/feat-x", s.BranchName)
	require.False(t, s.CreatedAt.Before(before))
}

func TestCreate_DuplicateReturnsError(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	_, err := store.Create(context.Background(), "feat-x", "feat/feat-x")
	require.NoError(t, err)

	_, err = store.Create(context.Background(), "feat-x", "feat/feat-x")
	require.ErrorIs(t, err, story.ErrStoryExists)
}

func TestCreate_EmptyName(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	_, err := store.Create(context.Background(), "", "feat/foo")
	require.Error(t, err)
}

func TestGet_ExistingStory(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	_, err := store.Create(context.Background(), "feat-x", "feat/feat-x")
	require.NoError(t, err)

	got, err := store.Get(context.Background(), "feat-x")
	require.NoError(t, err)
	require.Equal(t, "feat-x", got.Name)
	require.Equal(t, "feat/feat-x", got.BranchName)
}

func TestGet_UnknownStory(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	_, err := store.Get(context.Background(), "nonexistent")
	require.ErrorIs(t, err, story.ErrStoryNotFound)
}

func TestList_ReturnsSorted(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	_, err := store.Create(context.Background(), "zzz", "feat/zzz")
	require.NoError(t, err)
	_, err = store.Create(context.Background(), "aaa", "feat/aaa")
	require.NoError(t, err)
	_, err = store.Create(context.Background(), "mmm", "feat/mmm")
	require.NoError(t, err)

	list, err := store.List(context.Background())
	require.NoError(t, err)
	// List includes auto-created _default plus the 3 created stories.
	require.Len(t, list, 4)
	require.Equal(t, "_default", list[0].Name)
	require.Equal(t, "aaa", list[1].Name)
	require.Equal(t, "mmm", list[2].Name)
	require.Equal(t, "zzz", list[3].Name)
}

func TestDelete_RemovesFile(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	_, err := store.Create(context.Background(), "feat-x", "feat/feat-x")
	require.NoError(t, err)

	require.NoError(t, store.Delete(context.Background(), "feat-x"))

	_, err = store.Get(context.Background(), "feat-x")
	require.ErrorIs(t, err, story.ErrStoryNotFound)
}

func TestUpdate_AttachesProject(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	s, err := store.Create(context.Background(), "feat-x", "feat/feat-x")
	require.NoError(t, err)

	s.Projects = append(s.Projects, story.Project{
		Host:     testHost,
		Segments: []string{testOwner, testProject},
	})

	require.NoError(t, store.Update(context.Background(), s))

	got, err := store.Get(context.Background(), "feat-x")
	require.NoError(t, err)
	require.Len(t, got.Projects, 1)
	require.Equal(t, testHost, got.Projects[0].Host)
}

func TestUpdate_DuplicateProjectRejected(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	s, err := store.Create(context.Background(), "feat-x", "feat/feat-x")
	require.NoError(t, err)

	s.Projects = append(s.Projects, story.Project{
		Host:     testHost,
		Segments: []string{testOwner, testProject},
	})
	require.NoError(t, store.Update(context.Background(), s))

	// Try adding the same project again.
	got, err := store.Get(context.Background(), "feat-x")
	require.NoError(t, err)

	got.Projects = append(got.Projects, story.Project{
		Host:     testHost,
		Segments: []string{testOwner, testProject},
	})
	err = store.Update(context.Background(), got)
	require.ErrorIs(t, err, story.ErrProjectAlreadyAttached)
}

func TestUpdate_SequentialWritesSerialize(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	_, err := store.Create(context.Background(), "feat-x", "feat/feat-x")
	require.NoError(t, err)

	// Two sequential updates must both succeed without corruption.
	s, err := store.Get(context.Background(), "feat-x")
	require.NoError(t, err)

	s.Projects = append(s.Projects, story.Project{Host: testHost, Segments: []string{"a", "b"}})
	require.NoError(t, store.Update(context.Background(), s))

	s2, err := store.Get(context.Background(), "feat-x")
	require.NoError(t, err)

	s2.Projects = append(s2.Projects, story.Project{Host: "gitlab.com", Segments: []string{"c", "d"}})
	require.NoError(t, store.Update(context.Background(), s2))

	final, err := store.Get(context.Background(), "feat-x")
	require.NoError(t, err)
	require.Len(t, final.Projects, 2)
}

func TestDefaultStory_AutoCreated(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	list, err := store.List(context.Background())
	require.NoError(t, err)

	var found bool

	for _, s := range list {
		if s.Name == "_default" {
			found = true
		}
	}

	require.True(t, found, "_default story must be auto-created on first List")
}

func TestCreate_LockReleasedOnError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	storiesDir := filepath.Join(dir, "stories")
	require.NoError(t, os.MkdirAll(storiesDir, 0o700))

	// Make the directory read-only so writes fail.
	require.NoError(t, os.Chmod(storiesDir, 0o500)) //nolint:gosec // intentional permission test

	store := story.NewJSONStore(storiesDir)
	_, err := store.Create(context.Background(), "feat-x", "feat/feat-x")
	require.Error(t, err)

	// Restore permissions for cleanup.
	require.NoError(t, os.Chmod(storiesDir, 0o700)) //nolint:gosec // restoring permissions for test cleanup
}
