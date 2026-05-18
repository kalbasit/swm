package story_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"

	"github.com/kalbasit/swm/cmd/swm/internal/cli/story"
	"github.com/kalbasit/swm/cmd/swm/internal/config"
	"github.com/kalbasit/swm/cmd/swm/internal/hookexec"
)

var errHookFailed = errors.New("hook failed")

func TestCreateCmd_BasicCreation(t *testing.T) {
	t.Parallel()

	store := &stubStore{}
	cmd := story.NewCreateCmd(store, "", hookexec.Noop, config.DefaultBranchNameTemplate)

	cmd.SetArgs([]string{testStoryName})
	require.NoError(t, cmd.Execute())
	require.Equal(t, testStoryName, store.lastCreatedName)
	require.Equal(t, "feat/"+testStoryName, store.lastCreatedBranch)
}

func TestCreateCmd_CustomBranch(t *testing.T) {
	t.Parallel()

	store := &stubStore{}
	cmd := story.NewCreateCmd(store, "", hookexec.Noop, config.DefaultBranchNameTemplate)

	cmd.SetArgs([]string{"JIRA-42", "--branch", "fix/JIRA-42-crash"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "JIRA-42", store.lastCreatedName)
	require.Equal(t, "fix/JIRA-42-crash", store.lastCreatedBranch)
}

func TestCreateCmd_TemplateFromConfig(t *testing.T) {
	t.Parallel()

	store := &stubStore{}
	cmd := story.NewCreateCmd(store, "", hookexec.Noop, "fix/{{.Name}}")

	cmd.SetArgs([]string{testBugName})
	require.NoError(t, cmd.Execute())
	require.Equal(t, testBugName, store.lastCreatedName)
	require.Equal(t, "fix/"+testBugName, store.lastCreatedBranch)
}

func TestCreateCmd_BranchFlagOverridesTemplate(t *testing.T) {
	t.Parallel()

	store := &stubStore{}
	cmd := story.NewCreateCmd(store, "", hookexec.Noop, "fix/{{.Name}}")

	cmd.SetArgs([]string{testBugName, "--branch", "custom/branch"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "custom/branch", store.lastCreatedBranch)
}

func TestCreateCmd_InvalidTemplateErrors(t *testing.T) {
	t.Parallel()

	store := &stubStore{}
	cmd := story.NewCreateCmd(store, "", hookexec.Noop, "{{.Name")

	cmd.SetArgs([]string{testStoryName})
	err := cmd.Execute()
	require.Error(t, err)
	// No store write should have happened.
	require.Empty(t, store.lastCreatedName)
}

func TestCreateCmd_EmptyTemplateUsesDefault(t *testing.T) {
	t.Parallel()

	store := &stubStore{}
	cmd := story.NewCreateCmd(store, "", hookexec.Noop, "")

	cmd.SetArgs([]string{testStoryName})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "feat/"+testStoryName, store.lastCreatedBranch)
}

func TestCreateCmd_InvalidTemplate_NoHooksRun(t *testing.T) {
	t.Parallel()

	store := &stubStore{}

	var hooksCalled []string

	captureHook := hookexec.RunnerFunc(func(_ context.Context, cfg hookexec.RunConfig) error {
		hooksCalled = append(hooksCalled, cfg.Event)

		return nil
	})

	cmd := story.NewCreateCmd(store, "/code", captureHook, "{{.Name")
	cmd.SetArgs([]string{testStoryName})

	require.Error(t, cmd.Execute())
	require.Empty(t, hooksCalled)
}

func TestCreateCmd_Duplicate(t *testing.T) {
	t.Parallel()

	store := &stubStore{createErr: coreStory.ErrStoryExists}
	cmd := story.NewCreateCmd(store, "", hookexec.Noop, config.DefaultBranchNameTemplate)

	cmd.SetArgs([]string{testStoryName})
	require.Error(t, cmd.Execute())
}

func TestCreateCmd_PreHookAborts(t *testing.T) {
	t.Parallel()

	store := &stubStore{}

	var called []string

	captureHook := hookexec.RunnerFunc(func(_ context.Context, cfg hookexec.RunConfig) error {
		called = append(called, cfg.Event)
		if cfg.Event == "pre-story-create" {
			return errHookFailed
		}

		return nil
	})

	cmd := story.NewCreateCmd(store, "/code", captureHook, config.DefaultBranchNameTemplate)
	cmd.SetArgs([]string{testStoryName})

	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, called, "pre-story-create")
	// store.Create should not have been called.
	require.Empty(t, store.lastCreatedName)
}

func TestCreateCmd_HooksCalledInOrder(t *testing.T) {
	t.Parallel()

	store := &stubStore{}

	var called []string

	captureHook := hookexec.RunnerFunc(func(_ context.Context, cfg hookexec.RunConfig) error {
		called = append(called, cfg.Event)

		return nil
	})

	cmd := story.NewCreateCmd(store, "/code", captureHook, config.DefaultBranchNameTemplate)
	cmd.SetArgs([]string{testStoryName})

	require.NoError(t, cmd.Execute())
	require.Equal(t, []string{"pre-story-create", "post-story-create"}, called)
}
