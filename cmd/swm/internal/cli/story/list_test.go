package story_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"

	"github.com/kalbasit/swm/cmd/swm/internal/cli/story"
)

func TestListCmd_DefaultOnlyIsEmpty(t *testing.T) {
	t.Parallel()

	store := &stubStore{
		listStories: []*coreStory.Story{
			{Name: "_default"},
		},
	}

	cmd := story.NewListCmd(store, "_default")

	var out bytes.Buffer

	cmd.SetOut(&out)

	require.NoError(t, cmd.Execute())
	require.Empty(t, out.String())
}

func TestListCmd_MultipleStoriesExcludesDefault(t *testing.T) {
	t.Parallel()

	store := &stubStore{
		listStories: []*coreStory.Story{
			{Name: "_default"},
			{Name: "alpha"},
			{Name: "beta"},
		},
	}

	cmd := story.NewListCmd(store, "_default")

	var out bytes.Buffer

	cmd.SetOut(&out)

	require.NoError(t, cmd.Execute())
	require.Equal(t, "alpha\nbeta\n", out.String())
}

func TestListCmd_StoreError(t *testing.T) {
	t.Parallel()

	store := &stubStore{listErr: errHookFailed}

	cmd := story.NewListCmd(store, "_default")
	require.Error(t, cmd.Execute())
}
