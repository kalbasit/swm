## 1. SDK — Forge GRPCPlugin (sdk/go)

- [x] 1.1 Replace the stub in `sdk/go/forge/plugin.go` with a real `GRPCPlugin` struct (embedding `goplugin.NetRPCUnsupportedPlugin`, `GRPCClient` returning `pluginv1.NewForgeClient(conn)`, `GRPCServer` registering `pluginv1.RegisterForgeServer`); add `NewClient(conn *grpc.ClientConn) pluginv1.ForgeClient` constructor; update `Serve(impl Plugin)` to call `goplugin.Serve` with `"forge": &GRPCPlugin{Impl: impl}`; remove `ErrNotImplemented` (`sdk/go` module)
- [x] 1.2 Write unit tests in `sdk/go/forge/plugin_test.go` verifying the handshake config fields match `sdk/go/handshake` constants and that `GRPCPlugin` satisfies `goplugin.GRPCPlugin` (`sdk/go` module)
- [x] 1.3 Run `task sdk:test` → confirm exits 0

## 2. Plugin Manager — Wire Forge Capability (cmd/swm)

- [x] 2.1 Add `sdkforge "github.com/kalbasit/swm/sdk/go/forge"` import to `cmd/swm/internal/pluginmgr/manager.go`; update config to support `Forges []string` (a list); update `pluginSet` to handle forge plugins via `goplugin.PluginSet{"forge": &sdkforge.GRPCPlugin{}}`; add `GetForge(ctx, hostname string) (pluginv1.ForgeClient, error)` method that returns the client for the plugin claiming that hostname (`cmd/swm` module)
- [x] 2.2 Update `cmd/swm/internal/config/config.go` to add `Forges []string` field in `Plugins` struct and corresponding TOML key `forges`; update `Paths` map to accept forge plugin names (`cmd/swm` module)
- [x] 2.3 Run `task swm:test` → confirm exits 0

## 3. Hook Executor (cmd/swm)

- [x] 3.1 Create `cmd/swm/internal/hookexec/hookexec.go`: define `RunConfig` struct; implement `Run(ctx, cfg RunConfig) error` that searches the three tiers (global, per-repo, per-story) for executables under `<event>.d/`, runs them in lexical order, sets the required env vars (`SWM_HOOK`, `SWM_STORY`, `SWM_PROJECT_HOST`, `SWM_PROJECT_PATH`, `SWM_WORKTREE_PATH`, `SWM_REPO_PATH`), writes JSON on stdin in a goroutine, and aborts on non-zero for `pre-*` events / logs for `post-*` events (`cmd/swm` module)
- [x] 3.2 Write unit tests in `cmd/swm/internal/hookexec/hookexec_test.go`: cover no-hooks-exist, lexical ordering, pre-* abort, post-* log-and-continue, env vars set correctly, stdin JSON delivered; use a small Go binary in `testdata/fakehook/` that logs its env and stdin (`cmd/swm` module)
- [x] 3.3 Run `task swm:test` → confirm exits 0

## 4. forge-github Plugin (plugins/forge-github)

- [x] 4.1 Update `plugins/forge-github/go.mod`: add `github.com/kalbasit/swm/sdk/go`, `github.com/kalbasit/swm/proto`, `google.golang.org/grpc`, `github.com/google/go-github/v67`, `github.com/pelletier/go-toml/v2`; run `go mod tidy`; add `replace` directives for local modules; ensure `plugins/forge-github` is in `go.work` (`plugins/forge-github` module)
- [x] 4.2 Create `plugins/forge-github/internal/forge/github.go`: `GitHub` struct implementing `pluginv1.ForgeServer` with a `hostClient pluginv1.HostClient` field; `New(hostClient) *GitHub` constructor; implement `Info` returning `ForgeInfo{plugin_info: {name: "github", version: buildVersion}, hostnames: ["github.com"]}` (`plugins/forge-github` module)
- [x] 4.3 Implement `GitHub.tokenFromConfig(ctx) (string, error)`: call `hostClient.GetConfig` scoped to `"forge-github"`, unmarshal TOML to get `token_path` (default `~/.github_token`), expand `~`, read and trim the file, return `FailedPrecondition` if missing or empty (`plugins/forge-github` module)
- [x] 4.4 Implement `GitHub.ListPullRequests(req, stream)`: call `tokenFromConfig`, create a `github.NewClient` with the token, call `client.PullRequests.List(ctx, owner, repo, &github.PullRequestListOptions{State: "open"})`, stream one `PullRequest` message per result (`plugins/forge-github` module)
- [x] 4.5 Implement `GitHub.CreatePullRequest(ctx, req)`: call `tokenFromConfig`, create a `github.NewClient`, call `client.PullRequests.Create(ctx, owner, repo, &github.NewPullRequest{...})`, return the created `PullRequest` message; return `NotFound` if the repo doesn't exist, `FailedPrecondition` if token is missing (`plugins/forge-github` module)
- [x] 4.6 Implement `GitHub.GetPullRequest(ctx, req)`: call `tokenFromConfig`, fetch via `client.PullRequests.Get`, return `NotFound` if not found (`plugins/forge-github` module)
- [x] 4.7 Update `plugins/forge-github/main.go`: import `sdkforge` and `internal/forge`; call `sdkforge.Serve(forge.New(hostClient))` where `hostClient` is dialed from `SWM_HOST_SOCKET` env var; set `buildVersion` via ldflags (`plugins/forge-github` module)
- [x] 4.8 Write `plugins/forge-github/internal/forge/github_test.go`: use `go-github`'s `httptest` / `NewTestServerAndClient` helpers (or a mock HTTP server) to cover: `Info` returns correct hostnames, `ListPullRequests` success, `ListPullRequests` with no PRs, `CreatePullRequest` success, `CreatePullRequest` draft, `GetPullRequest` success, `GetPullRequest` not found, token missing returns `FailedPrecondition` (`plugins/forge-github` module)
- [x] 4.9 Run `task forge-github:test` → confirm exits 0

## 5. pr Commands (cmd/swm)

- [x] 5.1 Create `cmd/swm/internal/cli/pr/list.go`: `NewListCmd(cfg, store, mgr, resolver) *cobra.Command`; resolves story, iterates attached projects, calls `mgr.GetForge(ctx, project.Host)`, calls `forge.ListPullRequests`, prints `#<number>  <title>  <url>` per PR; skips projects with no registered forge silently (`cmd/swm` module)
- [x] 5.2 Create `cmd/swm/internal/cli/pr/create.go`: `NewCreateCmd(cfg, store, mgr, resolver) *cobra.Command` with `--title` (required), `--body`, `--base` (default `main`), `--draft`; calls `vcs.DetectProjectAtPath(cwd)` to get project ID; looks up forge via `mgr.GetForge(ctx, host)`; calls `forge.CreatePullRequest`; prints the PR URL (`cmd/swm` module)
- [x] 5.3 Register `swm pr list` and `swm pr create` under a new `swm pr` parent command in `cmd/swm/internal/cli/root.go` (or a new `pr.go` file); update `NewRootCmd` to add the `pr` subcommand (`cmd/swm` module)
- [x] 5.4 Write `cmd/swm/internal/cli/pr/list_test.go` and `create_test.go` with stub forge clients covering: list success, list no PRs, list no forge for host, create success, create draft, create no forge for host, create missing --title (`cmd/swm` module)
- [x] 5.5 Run `task swm:test` → confirm exits 0

## 6. Wire Hooks into Existing Commands (cmd/swm)

- [x] 6.1 Update `cmd/swm/internal/cli/story/create.go`: add `pre-story-create` hook before writing the story JSON and `post-story-create` hook after; pass hookexec.RunConfig with the story name; abort on pre-* failure (`cmd/swm` module)
- [x] 6.2 Update `cmd/swm/internal/cli/story/remove.go`: add `pre-story-remove` before removal, per-project `pre/post-worktree-remove` around each `vcs.RemoveWorktree` call, and `post-story-remove` after deletion (`cmd/swm` module)
- [x] 6.3 Update `cmd/swm/internal/cli/clone.go` (or wherever `swm clone` lives): add `pre-clone` before `vcs.Clone` and `post-clone` after (`cmd/swm` module)
- [x] 6.4 Update `cmd/swm/internal/cli/workspace/open.go`: add `pre-workspace-open` before the picker/session flow and `post-workspace-open` after (`cmd/swm` module)
- [x] 6.5 Update relevant unit tests to verify hooks are called at the correct points (use a `hookexec.NoopRunner` or a test double that captures calls) (`cmd/swm` module)
- [x] 6.6 Run `task swm:test` → confirm exits 0

## 7. Integration and Final Verification

- [x] 7.1 Add `plugins/forge-github` binary compilation to `cmd/swm/tests/integration/main_test.go`; write `TestPRListAndCreate` using a local HTTP test server (httptest) that stubs the GitHub API — verifying that `swm pr list` prints the correct output and `swm pr create` outputs the PR URL (`cmd/swm` module)
- [x] 7.2 Add a hook integration test `TestHooksRunOnStoryCreate` in integration_test.go: create a small executable hook script in the global hooks directory, run `swm story create`, verify the hook was executed (`cmd/swm` module)
- [x] 7.3 Run `task fmt` → confirm exits 0
- [x] 7.4 Run `task lint` → confirm exits 0
- [x] 7.5 Run `task test` → confirm exits 0
