package pr_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/cmd/swm/internal/cli/pr"
	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
)

const flagTitle = "--title"

// stubForgeClientCreate is a ForgeClient for create tests.
type stubForgeClientCreate struct {
	createErr error
	gotDraft  bool
}

func (c *stubForgeClientCreate) CreatePullRequest(
	_ context.Context,
	req *pluginv1.CreatePRRequest,
	_ ...grpc.CallOption,
) (*pluginv1.PullRequest, error) {
	c.gotDraft = req.GetDraft()

	if c.createErr != nil {
		return nil, c.createErr
	}

	return &pluginv1.PullRequest{
		Number: 42,
		Title:  req.GetTitle(),
		Url:    "https://github.com/o/r/pull/42",
	}, nil
}

func (c *stubForgeClientCreate) GetPullRequest(
	_ context.Context,
	_ *pluginv1.GetPRRequest,
	_ ...grpc.CallOption,
) (*pluginv1.PullRequest, error) {
	panic("stub")
}

func (c *stubForgeClientCreate) Info(
	_ context.Context,
	_ *pluginv1.Empty,
	_ ...grpc.CallOption,
) (*pluginv1.ForgeInfo, error) {
	panic("stub")
}

func (c *stubForgeClientCreate) ListPullRequests(
	_ context.Context,
	_ *pluginv1.ListPRsRequest,
	_ ...grpc.CallOption,
) (grpc.ServerStreamingClient[pluginv1.PullRequest], error) {
	panic("stub")
}

var _ pluginv1.ForgeClient = (*stubForgeClientCreate)(nil)

//nolint:paralleltest // t.Chdir changes process-wide CWD; not safe to run in parallel
func TestPRCreate_Success(t *testing.T) {
	codeRoot := t.TempDir()
	resolver := layout.NewResolver(codeRoot)

	repoDir := filepath.Join(codeRoot, "repositories", testGitHubHost, "o", "r")
	require.NoError(t, os.MkdirAll(repoDir, 0o750))
	t.Chdir(repoDir)

	forgeClient := &stubForgeClientCreate{}
	mgr := &stubForgeManager{forges: map[string]pluginv1.ForgeClient{
		testGitHubHost: forgeClient,
	}}

	var out bytes.Buffer

	cmd := pr.NewCreateCmd(mgr, resolver)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{flagTitle, "My PR", "--base", "main", "--head", "feat"})

	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "https://github.com/o/r/pull/42")
}

//nolint:paralleltest // t.Chdir changes process-wide CWD; not safe to run in parallel
func TestPRCreate_Draft(t *testing.T) {
	codeRoot := t.TempDir()
	resolver := layout.NewResolver(codeRoot)

	repoDir := filepath.Join(codeRoot, "repositories", testGitHubHost, "o", "r")
	require.NoError(t, os.MkdirAll(repoDir, 0o750))
	t.Chdir(repoDir)

	forgeClient := &stubForgeClientCreate{}
	mgr := &stubForgeManager{forges: map[string]pluginv1.ForgeClient{
		testGitHubHost: forgeClient,
	}}

	var out bytes.Buffer

	cmd := pr.NewCreateCmd(mgr, resolver)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{flagTitle, "Draft PR", "--draft"})

	require.NoError(t, cmd.Execute())
	require.True(t, forgeClient.gotDraft, "expected draft=true")
}

//nolint:paralleltest // t.Chdir changes process-wide CWD; not safe to run in parallel
func TestPRCreate_NoForgeForHost(t *testing.T) {
	codeRoot := t.TempDir()
	resolver := layout.NewResolver(codeRoot)

	repoDir := filepath.Join(codeRoot, "repositories", "gitlab.com", "o", "r")
	require.NoError(t, os.MkdirAll(repoDir, 0o750))
	t.Chdir(repoDir)

	mgr := &stubForgeManager{forges: map[string]pluginv1.ForgeClient{}}

	var out bytes.Buffer

	cmd := pr.NewCreateCmd(mgr, resolver)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{flagTitle, "My PR"})

	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "no forge plugin")
}

func TestPRCreate_MissingTitle(t *testing.T) {
	t.Parallel()

	codeRoot := t.TempDir()
	resolver := layout.NewResolver(codeRoot)
	mgr := &stubForgeManager{}

	cmd := pr.NewCreateCmd(mgr, resolver)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.Error(t, err)
}
