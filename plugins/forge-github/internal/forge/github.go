// Package forge implements the swm Forge capability for github.com.
package forge

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/go-github/v67/github"
	"github.com/pelletier/go-toml/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"
)

// buildVersion is set via -ldflags at build time.
var buildVersion = "dev" //nolint:gochecknoglobals // set via ldflags at link time

type githubConfig struct {
	TokenPath string `toml:"token_path"`
}

// GitHub implements pluginv1.ForgeServer for github.com.
type GitHub struct {
	pluginv1.UnimplementedForgeServer
	hostClient pluginv1.HostClient
	// baseURL, when non-empty, overrides the GitHub API base URL (for tests).
	baseURL string
}

// New returns a GitHub forge server backed by the given host client.
func New(hostClient pluginv1.HostClient) *GitHub {
	return &GitHub{hostClient: hostClient}
}

// NewWithBaseURL returns a GitHub forge server with a custom API base URL (for tests).
func NewWithBaseURL(hostClient pluginv1.HostClient, baseURL string) *GitHub {
	return &GitHub{hostClient: hostClient, baseURL: baseURL}
}

// ownerRepo extracts the owner and repo from a ProjectID's segments.
func ownerRepo(id *pluginv1.ProjectID) (owner, repo string, err error) {
	if id == nil || len(id.GetSegments()) < 2 {
		return "", "", status.Error(codes.InvalidArgument, "project_id must have at least 2 path segments (owner/repo)")
	}

	return id.GetSegments()[0], id.GetSegments()[1], nil
}

// CreatePullRequest opens a new pull request and returns the created PR.
func (g *GitHub) CreatePullRequest(ctx context.Context, req *pluginv1.CreatePRRequest) (*pluginv1.PullRequest, error) {
	client, err := g.newGitHubClient(ctx)
	if err != nil {
		return nil, err
	}

	owner, repo, err := ownerRepo(req.GetProjectId())
	if err != nil {
		return nil, err
	}

	draft := req.GetDraft()

	pr, resp, err := client.PullRequests.Create(ctx, owner, repo, &github.NewPullRequest{
		Title: new(req.GetTitle()),
		Head:  new(req.GetHeadBranch()),
		Base:  new(req.GetBaseBranch()),
		Body:  new(req.GetBody()),
		Draft: new(draft),
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, status.Errorf(codes.NotFound, "repository %s/%s not found", owner, repo)
		}

		return nil, fmt.Errorf("creating pull request: %w", err)
	}

	return ghPRToProto(pr), nil
}

// GetPullRequest fetches a single pull request by number.
func (g *GitHub) GetPullRequest(ctx context.Context, req *pluginv1.GetPRRequest) (*pluginv1.PullRequest, error) {
	client, err := g.newGitHubClient(ctx)
	if err != nil {
		return nil, err
	}

	owner, repo, err := ownerRepo(req.GetProjectId())
	if err != nil {
		return nil, err
	}

	pr, resp, err := client.PullRequests.Get(ctx, owner, repo, int(req.GetNumber()))
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, status.Errorf(codes.NotFound, "pull request #%d not found in %s/%s", req.GetNumber(), owner, repo)
		}

		return nil, fmt.Errorf("getting pull request: %w", err)
	}

	return ghPRToProto(pr), nil
}

// Info returns metadata about this forge plugin and the hostnames it handles.
func (g *GitHub) Info(_ context.Context, _ *pluginv1.Empty) (*pluginv1.ForgeInfo, error) {
	return &pluginv1.ForgeInfo{
		PluginInfo: &pluginv1.PluginInfo{
			Name:    "github",
			Version: buildVersion,
		},
		ClaimedHosts: []string{"github.com"},
	}, nil
}

// ListPullRequests streams pull requests for the given project.
func (g *GitHub) ListPullRequests(req *pluginv1.ListPRsRequest, stream pluginv1.Forge_ListPullRequestsServer) error {
	ctx := stream.Context()

	client, err := g.newGitHubClient(ctx)
	if err != nil {
		return err
	}

	owner, repo, err := ownerRepo(req.GetProjectId())
	if err != nil {
		return err
	}

	var state string

	switch req.GetState() {
	case pluginv1.PullRequestFilter_PULL_REQUEST_FILTER_CLOSED:
		state = "closed"
	case pluginv1.PullRequestFilter_PULL_REQUEST_FILTER_ALL:
		state = "all"
	case pluginv1.PullRequestFilter_PULL_REQUEST_FILTER_UNSPECIFIED, pluginv1.PullRequestFilter_PULL_REQUEST_FILTER_OPEN:
		state = "open"
	}

	prs, _, err := client.PullRequests.List(ctx, owner, repo, &github.PullRequestListOptions{State: state})
	if err != nil {
		return fmt.Errorf("listing pull requests: %w", err)
	}

	for _, pr := range prs {
		if err := stream.Send(ghPRToProto(pr)); err != nil {
			return fmt.Errorf("sending pull request: %w", err)
		}
	}

	return nil
}

// newGitHubClient returns a GitHub API client authenticated with the user's token.
func (g *GitHub) newGitHubClient(ctx context.Context) (*github.Client, error) {
	token, err := g.tokenFromConfig(ctx)
	if err != nil {
		return nil, err
	}

	client := github.NewClient(nil).WithAuthToken(token)

	// g.baseURL takes precedence (unit tests); fall back to env var (integration tests).
	baseURL := g.baseURL
	if baseURL == "" {
		baseURL = os.Getenv("FORGE_GITHUB_API_URL")
	}

	if baseURL != "" {
		parsed, err := url.Parse(baseURL)
		if err != nil {
			return nil, fmt.Errorf("parsing GitHub base URL: %w", err)
		}

		client.BaseURL = parsed
	}

	return client, nil
}

// tokenFromConfig reads the GitHub token path from host config, then reads and returns the token.
func (g *GitHub) tokenFromConfig(ctx context.Context) (string, error) {
	if g.hostClient == nil {
		return "", status.Error(codes.FailedPrecondition, "no host client: GitHub token unavailable")
	}

	resp, err := g.hostClient.GetConfig(ctx, &pluginv1.GetConfigRequest{PluginName: "forge-github"})
	if err != nil {
		return "", fmt.Errorf("getting forge-github config: %w", err)
	}

	var cfg githubConfig
	if len(resp.GetToml()) > 0 {
		if err := toml.Unmarshal(resp.GetToml(), &cfg); err != nil {
			return "", fmt.Errorf("parsing forge-github config: %w", err)
		}
	}

	tokenPath := cfg.TokenPath
	if tokenPath == "" {
		tokenPath = "~/.github_token" //nolint:gosec // G101: default token file path, not a credential
	}

	if strings.HasPrefix(tokenPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolving home directory: %w", err)
		}

		tokenPath = filepath.Join(home, tokenPath[2:])
	}

	raw, err := os.ReadFile(tokenPath) //nolint:gosec // G304: path comes from trusted plugin config
	if err != nil {
		return "", status.Errorf(codes.FailedPrecondition, "reading GitHub token from %q: %v", tokenPath, err)
	}

	token := strings.TrimSpace(string(raw))
	if token == "" {
		return "", status.Errorf(codes.FailedPrecondition, "GitHub token file %q is empty", tokenPath)
	}

	return token, nil
}

// ghPRToProto converts a go-github PullRequest to a proto PullRequest message.
func ghPRToProto(pr *github.PullRequest) *pluginv1.PullRequest {
	return &pluginv1.PullRequest{
		Id:         strconv.Itoa(pr.GetNumber()),
		Number:     int64(pr.GetNumber()),
		Title:      pr.GetTitle(),
		Body:       pr.GetBody(),
		State:      prStateFromString(pr.GetState()),
		Url:        pr.GetHTMLURL(),
		HeadBranch: pr.GetHead().GetRef(),
		BaseBranch: pr.GetBase().GetRef(),
		Draft:      pr.GetDraft(),
	}
}

// prStateFromString maps a GitHub API state string to a PullRequestState enum value.
func prStateFromString(s string) pluginv1.PullRequestState {
	switch s {
	case "open":
		return pluginv1.PullRequestState_PULL_REQUEST_STATE_OPEN
	case "closed":
		return pluginv1.PullRequestState_PULL_REQUEST_STATE_CLOSED
	case "merged":
		return pluginv1.PullRequestState_PULL_REQUEST_STATE_MERGED
	default:
		return pluginv1.PullRequestState_PULL_REQUEST_STATE_UNSPECIFIED
	}
}
