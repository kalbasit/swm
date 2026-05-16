package story_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"

	"github.com/kalbasit/swm/cmd/swm/internal/cli/story"
)

func TestCreateCmd_BasicCreation(t *testing.T) {
	t.Parallel()

	store := &stubStore{}
	cmd := story.NewCreateCmd(store)

	cmd.SetArgs([]string{testStoryName})
	require.NoError(t, cmd.Execute())
	require.Equal(t, testStoryName, store.lastCreatedName)
	require.Equal(t, "feat/"+testStoryName, store.lastCreatedBranch)
}

func TestCreateCmd_CustomBranch(t *testing.T) {
	t.Parallel()

	store := &stubStore{}
	cmd := story.NewCreateCmd(store)

	cmd.SetArgs([]string{"JIRA-42", "--branch", "fix/JIRA-42-crash"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "JIRA-42", store.lastCreatedName)
	require.Equal(t, "fix/JIRA-42-crash", store.lastCreatedBranch)
}

func TestCreateCmd_Duplicate(t *testing.T) {
	t.Parallel()

	store := &stubStore{createErr: coreStory.ErrStoryExists}
	cmd := story.NewCreateCmd(store)

	cmd.SetArgs([]string{testStoryName})
	require.Error(t, cmd.Execute())
}
