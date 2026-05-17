package workspace_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"

	"github.com/kalbasit/swm/cmd/swm/internal/cli/workspace"
)

var errListStore = errors.New("store failure")

func TestListCmd(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		stories []*coreStory.Story
		want    string
	}{
		{
			name:    "empty store",
			stories: nil,
			want:    "",
		},
		{
			name:    "only default story is excluded",
			stories: []*coreStory.Story{{Name: testDefaultStory}},
			want:    "",
		},
		{
			name:    "workspace with no projects",
			stories: []*coreStory.Story{{Name: "feat-x"}},
			want:    "feat-x\n",
		},
		{
			name: "single workspace with one project",
			stories: []*coreStory.Story{
				{
					Name: "feat-x",
					Projects: []coreStory.Project{
						{Host: testHost, Segments: []string{"a", "b"}},
					},
				},
			},
			want: "feat-x\n└── github.com/a/b\n",
		},
		{
			name: "multiple workspaces sorted with multiple projects sorted",
			stories: []*coreStory.Story{
				{
					Name: testSortStoryAlpha,
					Projects: []coreStory.Project{
						{Host: testHost, Segments: []string{"c", "d"}},
						{Host: testHost, Segments: []string{"a", "b"}},
					},
				},
				{
					Name: testSortStoryBeta,
					Projects: []coreStory.Project{
						{Host: testHost, Segments: []string{"e", "f"}},
					},
				},
			},
			want: "alpha\n├── github.com/a/b\n└── github.com/c/d\nbeta\n└── github.com/e/f\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := &stubStore{listStories: tc.stories}
			cmd := workspace.NewListCmd(store, testDefaultStory)

			var out bytes.Buffer
			cmd.SetOut(&out)

			require.NoError(t, cmd.Execute())
			require.Equal(t, tc.want, out.String())
		})
	}
}

func TestListCmd_StoreError(t *testing.T) {
	t.Parallel()

	store := &stubStore{listErr: errListStore}
	cmd := workspace.NewListCmd(store, testDefaultStory)

	require.Error(t, cmd.Execute())
}

func TestListCmd_Integration(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := coreStory.NewJSONStore(dir)
	ctx := context.Background()

	_, err := store.Create(ctx, "story-1", "story-1")
	require.NoError(t, err)

	s1, err := store.Get(ctx, "story-1")
	require.NoError(t, err)

	s1.Projects = append(s1.Projects, coreStory.Project{Host: testHost, Segments: []string{"a", "b"}})
	require.NoError(t, store.Update(ctx, s1))

	_, err = store.Create(ctx, "story-2", "story-2")
	require.NoError(t, err)

	s2, err := store.Get(ctx, "story-2")
	require.NoError(t, err)

	s2.Projects = append(
		s2.Projects,
		coreStory.Project{Host: testHost, Segments: []string{"e", "f"}},
		coreStory.Project{Host: testHost, Segments: []string{"c", "d"}},
	)
	require.NoError(t, store.Update(ctx, s2))

	cmd := workspace.NewListCmd(store, testDefaultStory)

	var out bytes.Buffer
	cmd.SetOut(&out)

	require.NoError(t, cmd.Execute())

	want := "story-1\n└── github.com/a/b\nstory-2\n├── github.com/c/d\n└── github.com/e/f\n"
	require.Equal(t, want, out.String())
}
