package pr_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	coreStory "github.com/kalbasit/swm/cmd/swm/internal/core/story"
	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/cmd/swm/internal/cli/pr"
	"github.com/kalbasit/swm/cmd/swm/internal/config"
)

const (
	testPRStoryName  = "feat-x"
	testGitHubHost   = "github.com"
	testDefaultStory = "_default"
	flagStory        = "--story"
)

// errNoForge is returned by stubForgeManager when no forge is configured.
var errNoForge = errors.New("no forge configured")

// stubForgeManager implements the forgeManager interface for tests.
type stubForgeManager struct {
	// forges maps hostname -> forge client.
	forges map[string]pluginv1.ForgeClient
}

func (m *stubForgeManager) GetForge(_ context.Context, hostname string) (pluginv1.ForgeClient, error) {
	if f, ok := m.forges[hostname]; ok {
		return f, nil
	}

	return nil, fmt.Errorf("%w: %q", errNoForge, hostname)
}

// stubForgeClient is a minimal ForgeClient for tests.
type stubForgeClient struct {
	prs     []*pluginv1.PullRequest
	listErr error
}

func (c *stubForgeClient) CreatePullRequest(
	_ context.Context,
	_ *pluginv1.CreatePRRequest,
	_ ...grpc.CallOption,
) (*pluginv1.PullRequest, error) {
	panic("stub")
}

func (c *stubForgeClient) GetPullRequest(
	_ context.Context,
	_ *pluginv1.GetPRRequest,
	_ ...grpc.CallOption,
) (*pluginv1.PullRequest, error) {
	panic("stub")
}

func (c *stubForgeClient) Info(
	_ context.Context,
	_ *pluginv1.Empty,
	_ ...grpc.CallOption,
) (*pluginv1.ForgeInfo, error) {
	panic("stub")
}

func (c *stubForgeClient) ListPullRequests(
	_ context.Context,
	_ *pluginv1.ListPRsRequest,
	_ ...grpc.CallOption,
) (grpc.ServerStreamingClient[pluginv1.PullRequest], error) {
	if c.listErr != nil {
		return nil, c.listErr
	}

	return &stubPRStream{prs: c.prs}, nil
}

var _ pluginv1.ForgeClient = (*stubForgeClient)(nil)

// stubPRStream is a grpc.ServerStreamingClient[PullRequest] that returns a fixed list.
type stubPRStream struct {
	prs []*pluginv1.PullRequest
	idx int
}

func (s *stubPRStream) CloseSend() error             { return nil }
func (s *stubPRStream) Context() context.Context     { return context.Background() }
func (s *stubPRStream) Header() (metadata.MD, error) { panic("stub") }

func (s *stubPRStream) Recv() (*pluginv1.PullRequest, error) {
	if s.idx >= len(s.prs) {
		return nil, io.EOF
	}

	pr := s.prs[s.idx]
	s.idx++

	return pr, nil
}

func (s *stubPRStream) RecvMsg(any) error    { panic("stub") }
func (s *stubPRStream) SendMsg(any) error    { panic("stub") }
func (s *stubPRStream) Trailer() metadata.MD { return nil }

// stubStore is a minimal coreStory.Store for tests.
type stubStore struct {
	story *coreStory.Story
	err   error
}

func (s *stubStore) Create(_ context.Context, _, _ string) (*coreStory.Story, error) {
	panic("stub")
}

func (s *stubStore) Delete(_ context.Context, _ string) error { panic("stub") }

func (s *stubStore) Get(_ context.Context, _ string) (*coreStory.Story, error) {
	if s.err != nil {
		return nil, s.err
	}

	return s.story, nil
}

func (s *stubStore) List(_ context.Context) ([]*coreStory.Story, error) { panic("stub") }

func (s *stubStore) Update(_ context.Context, _ *coreStory.Story) error { panic("stub") }

var _ coreStory.Store = (*stubStore)(nil)

func TestPRList_Success(t *testing.T) {
	t.Parallel()

	prs := []*pluginv1.PullRequest{
		{Number: 1, Title: "Fix bug", Url: "https://github.com/o/r/pull/1"},
		{Number: 2, Title: "Add feature", Url: "https://github.com/o/r/pull/2"},
	}

	store := &stubStore{story: &coreStory.Story{
		Name:     testPRStoryName,
		Projects: []coreStory.Project{{Host: testGitHubHost, Segments: []string{"o", "r"}}},
	}}

	mgr := &stubForgeManager{forges: map[string]pluginv1.ForgeClient{
		testGitHubHost: &stubForgeClient{prs: prs},
	}}

	cfg := &config.Config{DefaultStory: testDefaultStory}

	var out bytes.Buffer

	cmd := pr.NewListCmd(store, mgr, cfg)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{flagStory, testPRStoryName})

	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "Fix bug")
	require.Contains(t, out.String(), "Add feature")
	require.Contains(t, out.String(), "https://github.com/o/r/pull/1")
}

func TestPRList_NoPRs(t *testing.T) {
	t.Parallel()

	store := &stubStore{story: &coreStory.Story{
		Name:     testPRStoryName,
		Projects: []coreStory.Project{{Host: testGitHubHost, Segments: []string{"o", "r"}}},
	}}

	mgr := &stubForgeManager{forges: map[string]pluginv1.ForgeClient{
		testGitHubHost: &stubForgeClient{prs: nil},
	}}

	cfg := &config.Config{DefaultStory: testDefaultStory}

	var out bytes.Buffer

	cmd := pr.NewListCmd(store, mgr, cfg)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{flagStory, testPRStoryName})

	require.NoError(t, cmd.Execute())
	require.Empty(t, out.String())
}

func TestPRList_NoForgeForHost(t *testing.T) {
	t.Parallel()

	store := &stubStore{story: &coreStory.Story{
		Name: testPRStoryName,
		// Project on gitlab.com — no forge configured.
		Projects: []coreStory.Project{{Host: "gitlab.com", Segments: []string{"o", "r"}}},
	}}

	// No forges configured.
	mgr := &stubForgeManager{forges: map[string]pluginv1.ForgeClient{}}

	cfg := &config.Config{DefaultStory: testDefaultStory}

	var out bytes.Buffer

	cmd := pr.NewListCmd(store, mgr, cfg)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{flagStory, testPRStoryName})

	// Should succeed silently (skip unknown hosts).
	require.NoError(t, cmd.Execute())
	require.Empty(t, out.String())
}

func TestPRList_FallsBackToDefaultStory(t *testing.T) {
	t.Parallel()

	store := &stubStore{story: &coreStory.Story{
		Name:     testDefaultStory,
		Projects: []coreStory.Project{{Host: testGitHubHost, Segments: []string{"o", "r"}}},
	}}

	mgr := &stubForgeManager{forges: map[string]pluginv1.ForgeClient{
		testGitHubHost: &stubForgeClient{prs: []*pluginv1.PullRequest{
			{Number: 1, Title: "Default PR", Url: "https://github.com/o/r/pull/1"},
		}},
	}}

	cfg := &config.Config{DefaultStory: testDefaultStory}

	var out bytes.Buffer

	cmd := pr.NewListCmd(store, mgr, cfg)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{}) // no --story flag, no $SWM_STORY

	require.NoError(t, cmd.Execute())
	require.Contains(t, out.String(), "Default PR")
}
