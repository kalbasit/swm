package forge_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"

	"github.com/kalbasit/swm/plugins/forge-github/internal/forge"
)

const (
	testGitHubHost = "github.com"
	testOwner      = "owner"
	testRepo       = "repo"
	testToken      = "ghp_test_token" //nolint:gosec // G101: test placeholder, not a real credential
	testBaseBranch = "main"
)

// fakeHostClient implements pluginv1.HostClient for tests.
type fakeHostClient struct {
	toml []byte
}

func (c *fakeHostClient) CallCapability(
	_ context.Context,
	_ *pluginv1.CallCapabilityRequest,
	_ ...grpc.CallOption,
) (*pluginv1.CallCapabilityResponse, error) {
	panic("stub")
}

func (c *fakeHostClient) GetCodeRoot(
	_ context.Context,
	_ *pluginv1.Empty,
	_ ...grpc.CallOption,
) (*pluginv1.PathResponse, error) {
	panic("stub")
}

func (c *fakeHostClient) GetConfig(
	_ context.Context,
	_ *pluginv1.GetConfigRequest,
	_ ...grpc.CallOption,
) (*pluginv1.Config, error) {
	return &pluginv1.Config{Toml: c.toml}, nil
}

func (c *fakeHostClient) GetCurrentStory(
	_ context.Context,
	_ *pluginv1.Empty,
	_ ...grpc.CallOption,
) (*pluginv1.Story, error) {
	panic("stub")
}

func (c *fakeHostClient) ListProjects(
	_ context.Context,
	_ *pluginv1.ListProjectsRequest,
	_ ...grpc.CallOption,
) (grpc.ServerStreamingClient[pluginv1.Project], error) {
	panic("stub")
}

func (c *fakeHostClient) Log(
	_ context.Context,
	_ *pluginv1.LogRequest,
	_ ...grpc.CallOption,
) (*pluginv1.Empty, error) {
	panic("stub")
}

// fakeListStream captures PR messages sent via ListPullRequests.
type fakeListStream struct {
	ctx context.Context
	prs []*pluginv1.PullRequest
}

func (s *fakeListStream) Context() context.Context { return s.ctx }
func (s *fakeListStream) RecvMsg(any) error        { return nil }
func (s *fakeListStream) Send(pr *pluginv1.PullRequest) error {
	s.prs = append(s.prs, pr)

	return nil
}

func (s *fakeListStream) SendHeader(metadata.MD) error { return nil }
func (s *fakeListStream) SendMsg(any) error            { return nil }
func (s *fakeListStream) SetHeader(metadata.MD) error  { return nil }
func (s *fakeListStream) SetTrailer(metadata.MD)       {}

// writeTokenFile writes a token to a temp file and returns its path.
func writeTokenFile(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "token")

	require.NoError(t, os.WriteFile(path, []byte(testToken), 0o600))

	return path
}

// prJSON returns a minimal JSON representation of a GitHub pull request.
func prJSON(number int, title, state, htmlURL, head string, draft bool) map[string]any {
	return map[string]any{
		"number":   number,
		"title":    title,
		"body":     "body of " + title,
		"state":    state,
		"html_url": htmlURL,
		"draft":    draft,
		"head":     map[string]any{"ref": head},
		"base":     map[string]any{"ref": testBaseBranch},
	}
}

func TestGitHub_Info(t *testing.T) {
	t.Parallel()

	g := forge.New(nil)

	info, err := g.Info(context.Background(), &pluginv1.Empty{})
	require.NoError(t, err)
	require.Equal(t, "github", info.GetPluginInfo().GetName())
	require.Equal(t, []string{testGitHubHost}, info.GetClaimedHosts())
}

func TestGitHub_ListPullRequests_Success(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/repos/owner/repo/pulls", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		prs := []map[string]any{
			prJSON(1, "Fix bug", "open", "https://github.com/owner/repo/pull/1", "fix-bug", false),
			prJSON(2, "Add feature", "open", "https://github.com/owner/repo/pull/2", "feat", false),
		}

		//nolint:errcheck // test mock, response write failure is non-critical
		_ = json.NewEncoder(w).Encode(prs)
	})

	tokenFile := writeTokenFile(t)
	hc := &fakeHostClient{toml: fmt.Appendf(nil, "token_path = %q", tokenFile)}
	g := forge.NewWithBaseURL(hc, server.URL+"/")

	stream := &fakeListStream{ctx: context.Background()}
	err := g.ListPullRequests(&pluginv1.ListPRsRequest{
		ProjectId: &pluginv1.ProjectID{Host: testGitHubHost, Segments: []string{testOwner, testRepo}},
	}, stream)

	require.NoError(t, err)
	require.Len(t, stream.prs, 2)
	require.Equal(t, "Fix bug", stream.prs[0].GetTitle())
	require.Equal(t, int64(1), stream.prs[0].GetNumber())
	require.Equal(t, "Add feature", stream.prs[1].GetTitle())
}

func TestGitHub_ListPullRequests_NoPRs(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/repos/owner/repo/pulls", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		//nolint:errcheck // test mock, response write failure is non-critical
		fmt.Fprintln(w, "[]")
	})

	tokenFile := writeTokenFile(t)
	hc := &fakeHostClient{toml: fmt.Appendf(nil, "token_path = %q", tokenFile)}
	g := forge.NewWithBaseURL(hc, server.URL+"/")

	stream := &fakeListStream{ctx: context.Background()}
	err := g.ListPullRequests(&pluginv1.ListPRsRequest{
		ProjectId: &pluginv1.ProjectID{Host: testGitHubHost, Segments: []string{testOwner, testRepo}},
	}, stream)

	require.NoError(t, err)
	require.Empty(t, stream.prs)
}

func TestGitHub_CreatePullRequest_Success(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/repos/owner/repo/pulls", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		pr := prJSON(42, "My PR", "open", "https://github.com/owner/repo/pull/42", "feature", false)

		//nolint:errcheck // test mock, response write failure is non-critical
		_ = json.NewEncoder(w).Encode(pr)
	})

	tokenFile := writeTokenFile(t)
	hc := &fakeHostClient{toml: fmt.Appendf(nil, "token_path = %q", tokenFile)}
	g := forge.NewWithBaseURL(hc, server.URL+"/")

	pr, err := g.CreatePullRequest(context.Background(), &pluginv1.CreatePRRequest{
		ProjectId:  &pluginv1.ProjectID{Host: testGitHubHost, Segments: []string{testOwner, testRepo}},
		Title:      "My PR",
		HeadBranch: "feature",
		BaseBranch: testBaseBranch,
	})

	require.NoError(t, err)
	require.Equal(t, int64(42), pr.GetNumber())
	require.Equal(t, "My PR", pr.GetTitle())
	require.Equal(t, "https://github.com/owner/repo/pull/42", pr.GetUrl())
}

func TestGitHub_CreatePullRequest_Draft(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	var gotDraft bool

	mux.HandleFunc("/repos/owner/repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any

		//nolint:errcheck // test mock, decode failure is non-critical
		_ = json.NewDecoder(r.Body).Decode(&body)

		if d, ok := body["draft"].(bool); ok {
			gotDraft = d
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		pr := prJSON(7, "Draft PR", "open", "https://github.com/owner/repo/pull/7", "draft-branch", true)

		//nolint:errcheck // test mock, response write failure is non-critical
		_ = json.NewEncoder(w).Encode(pr)
	})

	tokenFile := writeTokenFile(t)
	hc := &fakeHostClient{toml: fmt.Appendf(nil, "token_path = %q", tokenFile)}
	g := forge.NewWithBaseURL(hc, server.URL+"/")

	pr, err := g.CreatePullRequest(context.Background(), &pluginv1.CreatePRRequest{
		ProjectId:  &pluginv1.ProjectID{Host: testGitHubHost, Segments: []string{testOwner, testRepo}},
		Title:      "Draft PR",
		HeadBranch: "draft-branch",
		BaseBranch: testBaseBranch,
		Draft:      true,
	})

	require.NoError(t, err)
	require.True(t, gotDraft, "expected draft=true in request body")
	require.True(t, pr.GetDraft())
}

func TestGitHub_GetPullRequest_Success(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/repos/owner/repo/pulls/5", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		pr := prJSON(5, "PR five", "closed", "https://github.com/owner/repo/pull/5", "pr-five", false)

		//nolint:errcheck // test mock, response write failure is non-critical
		_ = json.NewEncoder(w).Encode(pr)
	})

	tokenFile := writeTokenFile(t)
	hc := &fakeHostClient{toml: fmt.Appendf(nil, "token_path = %q", tokenFile)}
	g := forge.NewWithBaseURL(hc, server.URL+"/")

	pr, err := g.GetPullRequest(context.Background(), &pluginv1.GetPRRequest{
		ProjectId: &pluginv1.ProjectID{Host: testGitHubHost, Segments: []string{testOwner, testRepo}},
		Number:    5,
	})

	require.NoError(t, err)
	require.Equal(t, int64(5), pr.GetNumber())
	require.Equal(t, "PR five", pr.GetTitle())
	require.Equal(t, pluginv1.PullRequestState_PULL_REQUEST_STATE_CLOSED, pr.GetState())
}

func TestGitHub_GetPullRequest_NotFound(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/repos/owner/repo/pulls/999", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)

		//nolint:errcheck // test mock, response write failure is non-critical
		fmt.Fprintln(w, `{"message":"Not Found"}`)
	})

	tokenFile := writeTokenFile(t)
	hc := &fakeHostClient{toml: fmt.Appendf(nil, "token_path = %q", tokenFile)}
	g := forge.NewWithBaseURL(hc, server.URL+"/")

	_, err := g.GetPullRequest(context.Background(), &pluginv1.GetPRRequest{
		ProjectId: &pluginv1.ProjectID{Host: testGitHubHost, Segments: []string{testOwner, testRepo}},
		Number:    999,
	})

	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestGitHub_GetPullRequest_Merged(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/repos/owner/repo/pulls/10", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		pr := prJSON(10, "Merged PR", "closed", "https://github.com/owner/repo/pull/10", "merged-branch", false)
		pr["merged"] = true

		//nolint:errcheck // test mock, response write failure is non-critical
		_ = json.NewEncoder(w).Encode(pr)
	})

	tokenFile := writeTokenFile(t)
	hc := &fakeHostClient{toml: fmt.Appendf(nil, "token_path = %q", tokenFile)}
	g := forge.NewWithBaseURL(hc, server.URL+"/")

	pr, err := g.GetPullRequest(context.Background(), &pluginv1.GetPRRequest{
		ProjectId: &pluginv1.ProjectID{Host: testGitHubHost, Segments: []string{testOwner, testRepo}},
		Number:    10,
	})

	require.NoError(t, err)
	require.Equal(t, int64(10), pr.GetNumber())
	require.Equal(t, pluginv1.PullRequestState_PULL_REQUEST_STATE_MERGED, pr.GetState())
}

func TestGitHub_TokenMissing_FailedPrecondition(t *testing.T) {
	t.Parallel()

	// Point to a token file that does not exist.
	hc := &fakeHostClient{toml: []byte(`token_path = "/nonexistent/path/to/token"`)}
	g := forge.New(hc)

	stream := &fakeListStream{ctx: context.Background()}
	err := g.ListPullRequests(&pluginv1.ListPRsRequest{
		ProjectId: &pluginv1.ProjectID{Host: testGitHubHost, Segments: []string{testOwner, testRepo}},
	}, stream)

	require.Error(t, err)
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}
