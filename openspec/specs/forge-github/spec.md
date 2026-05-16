### Requirement: sdk/go/forge GRPCPlugin wiring
`sdk/go/forge` SHALL provide a real `GRPCPlugin` struct (embedding `goplugin.NetRPCUnsupportedPlugin`) with `GRPCClient` returning `pluginv1.NewForgeClient(conn)` and `GRPCServer` registering `pluginv1.RegisterForgeServer`. It SHALL provide `Serve(impl Plugin)` calling `goplugin.Serve` with `"forge": &GRPCPlugin{Impl: impl}` and `NewClient(conn *grpc.ClientConn) pluginv1.ForgeClient`. The pattern SHALL mirror `sdk/go/picker`.

#### Scenario: GRPCPlugin satisfies goplugin.GRPCPlugin interface
- **WHEN** the sdk/go/forge package is compiled
- **THEN** `var _ goplugin.GRPCPlugin = (*forge.GRPCPlugin)(nil)` compiles without error

#### Scenario: NewClient returns a ForgeClient
- **WHEN** `forge.NewClient(conn)` is called with a gRPC connection
- **THEN** it returns a value satisfying `pluginv1.ForgeClient`

### Requirement: pluginmgr forge capability
The plugin manager SHALL support a `forges` list in config. For each named forge plugin it SHALL launch the binary, call `Info()`, and register the plugin's claimed hostnames in an internal `hostname → ForgeClient` map. `mgr.GetForge(ctx, hostname)` SHALL return the `ForgeClient` for that hostname or an error if none is registered.

#### Scenario: Single forge plugin registered for github.com
- **WHEN** config has `forges = ["github"]` and the `forge-github` binary is available
- **THEN** `mgr.GetForge(ctx, "github.com")` returns the `ForgeClient` without error

#### Scenario: No forge plugin registered
- **WHEN** config has no `forges` entry
- **THEN** `mgr.GetForge(ctx, "github.com")` returns an error

### Requirement: forge-github plugin Info
`forge-github` SHALL implement `Forge.Info(Empty) → ForgeInfo`. The response SHALL have `plugin_info.name = "github"`, `plugin_info.version` set from build-time ldflags, and `hostnames = ["github.com"]`.

#### Scenario: Info returns correct hostname
- **WHEN** `forge-github.Info` is called
- **THEN** the response contains `hostnames = ["github.com"]` and `plugin_info.name = "github"`

### Requirement: forge-github ListPullRequests
`forge-github` SHALL implement `Forge.ListPullRequests(ListPRsRequest) → stream PullRequest`. The request SHALL carry a `ProjectID`. The plugin SHALL list open PRs from the GitHub API for the repository identified by `host/segments[0]/segments[1]`. Each streamed `PullRequest` SHALL have `number`, `title`, `url`, `state`, `head_branch`, `base_branch`, and `author`.

#### Scenario: Successful PR listing
- **WHEN** `ListPullRequests` is called with a valid project ID and the GitHub API returns open PRs
- **THEN** one `PullRequest` message is streamed per open PR, each with `number`, `title`, `url`, `state`, `head_branch`, `base_branch`, and `author` populated

#### Scenario: Repository with no open PRs
- **WHEN** `ListPullRequests` is called and the repository has no open PRs
- **THEN** the stream completes with zero messages and no error

#### Scenario: Missing token
- **WHEN** `ListPullRequests` is called and the token file does not exist or is empty
- **THEN** the RPC returns a gRPC `FailedPrecondition` error

### Requirement: forge-github CreatePullRequest
`forge-github` SHALL implement `Forge.CreatePullRequest(CreatePRRequest) → PullRequest`. The request SHALL carry `project_id`, `title`, `body`, `head_branch`, `base_branch`, and `draft` (bool). The plugin SHALL create the PR via the GitHub API and return the created `PullRequest` with `number`, `title`, `url`, `state`, `head_branch`, `base_branch`, and `author`.

#### Scenario: Successful PR creation
- **WHEN** `CreatePullRequest` is called with valid fields and the GitHub API succeeds
- **THEN** the response contains the created PR with `number > 0`, `url` set, and `state = "open"`

#### Scenario: Draft PR
- **WHEN** `CreatePullRequest` is called with `draft = true`
- **THEN** the PR is created as a draft (GitHub API `draft: true`)

#### Scenario: Missing token
- **WHEN** `CreatePullRequest` is called and the token file does not exist
- **THEN** the RPC returns a gRPC `FailedPrecondition` error

### Requirement: forge-github GetPullRequest
`forge-github` SHALL implement `Forge.GetPullRequest(GetPRRequest) → PullRequest`. The request SHALL carry `project_id` and `number`. The plugin SHALL fetch the PR from the GitHub API and return it.

#### Scenario: Successful PR retrieval
- **WHEN** `GetPullRequest` is called with a valid project ID and PR number
- **THEN** the response contains the PR with all fields populated

#### Scenario: PR not found
- **WHEN** `GetPullRequest` is called with a PR number that does not exist
- **THEN** the RPC returns a gRPC `NotFound` error

### Requirement: forge-github token loading
The plugin SHALL read the GitHub token from the file path configured at `plugins.config.forge-github.token_path` (obtained via `host.GetConfig`). If `token_path` is not configured, the plugin SHALL default to `~/.github_token`. The file SHALL be read at call time (not at startup). Leading/trailing whitespace SHALL be trimmed.

#### Scenario: Token loaded from configured path
- **WHEN** `token_path = "~/.secrets/gh_token"` is set in config and the file contains a valid token
- **THEN** the plugin uses that token for GitHub API requests

#### Scenario: Default token path used
- **WHEN** `token_path` is not configured and `~/.github_token` exists
- **THEN** the plugin reads the token from `~/.github_token`
