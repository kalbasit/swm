## 1. SDK Transport Layer (sdk/go)

- [x] 1.1 Add `GRPCPlugin` struct to `sdk/go/session/plugin.go` implementing `go-plugin`'s `plugin.GRPCPlugin` interface; update `Serve(impl Plugin) error` to call `plugin.Serve` with the registered plugin map; add `NewClient(conn *grpc.ClientConn) Plugin` constructor returning a `pluginv1.SessionClient` wrapper (`sdk/go` module)
- [x] 1.2 Add `GRPCPlugin` struct to `sdk/go/vcs/plugin.go` with same pattern as 1.1; update `Serve` and add `NewClient` (`sdk/go` module)
- [x] 1.3 Write unit tests for `sdk/go/session` and `sdk/go/vcs` verifying the handshake config fields match `sdk/go/handshake` constants (`sdk/go` module)
- [x] 1.4 Run `task sdk:test` → confirm exits 0

## 2. Host Core — Config (cmd/swm)

- [x] 2.1 Add deps to `cmd/swm/go.mod`: `pelletier/go-toml/v2`, `github.com/adrg/xdg`, `github.com/gofrs/flock`, `github.com/hashicorp/go-plugin`; run `go mod tidy` (`cmd/swm` module)
- [x] 2.2 Create `cmd/swm/internal/config/config.go`: `Config` struct with fields `CodeRoot`, `DefaultStory`, `Plugins` (session/vcs/picker/forges names), `PluginPaths` (map), `PluginConfig` (map of raw TOML); apply defaults (`code_root = "~/code"`, `default_story = "_default"`) before unmarshal (`cmd/swm` module)
- [x] 2.3 Create `cmd/swm/internal/config/load.go`: `Load(path string) (*Config, error)` reading the TOML file via `pelletier/go-toml/v2`; return sentinel `ErrConfigNotFound` if path doesn't exist so callers can apply defaults (`cmd/swm` module)
- [x] 2.4 Write table-driven tests in `cmd/swm/internal/config/config_test.go` covering: file not found (returns defaults), all fields set, missing optional fields, bad TOML syntax (`cmd/swm` module)

## 3. Host Core — Story Store (cmd/swm)

- [x] 3.1 Create `cmd/swm/internal/core/story/story.go`: `Story` struct (name, branch_name, created_at, vcs, projects, metadata) with JSON tags (`snake_case`); `Project` struct (host, segments, vcs, attached_at); sentinel errors `ErrStoryExists`, `ErrStoryNotFound`, `ErrProjectAlreadyAttached` (`cmd/swm` module)
- [x] 3.2 Create `cmd/swm/internal/core/story/store.go`: `Store` interface with `Create`, `Get`, `List`, `Delete`, `Update` methods; `JSONStore` struct implementing it with XDG data dir + flock on all writes (`cmd/swm` module)
- [x] 3.3 Implement `JSONStore.Create`: validate name non-empty, check for existing file, acquire flock, write JSON, release flock; run `post-create` default story bootstrap on first access (`cmd/swm` module)
- [x] 3.4 Implement `JSONStore.Get`, `List`, `Delete`, `Update` with appropriate flock semantics; `List` reads all `*.json` files and returns sorted by name (`cmd/swm` module)
- [x] 3.5 Write table-driven tests in `cmd/swm/internal/core/story/store_test.go` using a temp XDG dir; cover all CRUD operations + duplicate create + flock (simulate via two sequential writes) (`cmd/swm` module)

## 4. Host Core — Layout Resolver (cmd/swm)

- [x] 4.1 Create `cmd/swm/internal/core/layout/layout.go`: `Resolver` struct with `CodeRoot` field; methods `CanonicalPath(projectID) string` → `<code_root>/repositories/<host>/<seg1>/.../<segN>` and `WorktreePath(storyName, projectID) string` → `<code_root>/stories/<story>/<host>/<seg1>/.../<segN>` (`cmd/swm` module)
- [x] 4.2 Write unit tests in `cmd/swm/internal/core/layout/layout_test.go` covering multi-segment project IDs and the examples from TDD §5.1 (`cmd/swm` module)

## 5. Plugin Manager (cmd/swm)

- [x] 5.1 Create `cmd/swm/internal/pluginmgr/manager.go`: `Manager` struct holding a capability registry (map of capability name → launched client), config reference, and host server address; `New(cfg *config.Config, hostSocket string) *Manager` constructor (`cmd/swm` module)
- [x] 5.2 Implement `Manager.discover(capability, name) (string, error)`: search (1) explicit config paths, (2) `$XDG_DATA_HOME/swm/plugins/<name>/swm-plugin-<capability>-<name>`, (3) `exec.LookPath("swm-plugin-<capability>-<name>")` (`cmd/swm` module)
- [x] 5.3 Implement `Manager.Get(capability string) (interface{}, error)`: lazy launch using go-plugin client; on first call: discover binary → create go-plugin client with `SWM_HOST_SOCKET` env var → call `Info()` → validate dep-graph → cache; return cached client on subsequent calls (`cmd/swm` module)
- [x] 5.4 Implement `Manager.Close() error`: call `Kill()` on all launched go-plugin clients; return joined errors (`cmd/swm` module)
- [x] 5.5 Write tests in `cmd/swm/internal/pluginmgr/manager_test.go` using a fake plugin binary (compiled in TestMain); cover: discovery (PATH), lazy launch, dep-graph validation failure, Close cleanup (`cmd/swm` module)

## 6. Host Services (cmd/swm)

- [x] 6.1 Create `cmd/swm/internal/hostsvc/server.go`: `Server` implementing `pluginv1.HostServer`; constructor takes `*config.Config`, layout resolver, and story store; starts a gRPC server on a temp Unix socket; exposes `SocketPath() string` (`cmd/swm` module)
- [x] 6.2 Implement `GetConfig(req)`: decode the `PluginConfig[req.plugin_name]` subtable and return as TOML bytes (`cmd/swm` module)
- [x] 6.3 Implement `GetCodeRoot(req)`: return `PathResponse{path: cfg.CodeRoot}` (`cmd/swm` module)
- [x] 6.4 Implement `ListProjects(req, stream)`: walk `<code_root>/repositories/`, identify project directories by the presence of VCS markers (`.git` initially), stream `Project` messages (`cmd/swm` module)
- [x] 6.5 Implement `Log(req)`: write the log entry to the host's `slog` logger at the appropriate level (`cmd/swm` module)
- [x] 6.6 Write tests in `cmd/swm/internal/hostsvc/server_test.go` using a real temp directory and gRPC client; cover GetConfig scoping, GetCodeRoot, ListProjects marker detection (`cmd/swm` module)

## 7. vcs-git Plugin (plugins/vcs-git)

- [x] 7.1 Add deps to `plugins/vcs-git/go.mod`: `github.com/kalbasit/swm/sdk/go`, `github.com/kalbasit/swm/proto`; run `go mod tidy`; add `replace` directives for local modules (`plugins/vcs-git` module)
- [x] 7.2 Create `plugins/vcs-git/internal/vcs/git.go`: `Git` struct implementing `pluginv1.VCSServer`; all methods shell out to system `git` via `os/exec`; no global state (`plugins/vcs-git` module)
- [x] 7.3 Implement `ParseRemoteURL`: regex/parse SSH, HTTPS, and git+ssh URL formats; strip `.git` suffix; return `ProjectID` or gRPC `InvalidArgument` (`plugins/vcs-git` module)
- [x] 7.4 Implement `Clone`: check for existing `.git` dir (return `AlreadyExists`), run `git clone <url> <canonical_path>`, capture stderr for errors (`plugins/vcs-git` module)
- [x] 7.5 Implement `CreateWorktree`: `os.MkdirAll` parent, run `git -C <canonical_path> worktree add [-b <branch>] <worktree_path> <branch>`; detect existing-branch vs new-branch case (`plugins/vcs-git` module)
- [x] 7.6 Implement `RemoveWorktree`: run `git -C <canonical_path> worktree remove --force <worktree_path>` then `git -C <canonical_path> worktree prune`; return `NotFound` if worktree not registered (`plugins/vcs-git` module)
- [x] 7.7 Implement `DetectProjectAtPath`: run `git -C <path> remote get-url origin`, pass result through ParseRemoteURL logic; return `NotFound` if not a git repo (`plugins/vcs-git` module)
- [x] 7.8 Implement `Info`: return `VCSInfo{plugin_info: {name: "git", version: buildVersion}, project_markers: [".git"]}` (`plugins/vcs-git` module)
- [x] 7.9 Update `plugins/vcs-git/main.go`: call `vcs.Serve(&Git{})` instead of `fmt.Println` stub; set `buildVersion` via ldflags (`plugins/vcs-git` module)
- [x] 7.10 Write unit tests in `plugins/vcs-git/internal/vcs/git_test.go` using a real temp git repo (created with `git init`); cover all five RPC methods + error cases (`plugins/vcs-git` module)
- [x] 7.11 Run `task vcs-git:test` → confirm exits 0

## 8. session-tmux Plugin (plugins/session-tmux)

- [x] 8.1 Add deps to `plugins/session-tmux/go.mod`: `github.com/kalbasit/swm/sdk/go`, `github.com/kalbasit/swm/proto`; run `go mod tidy`; add `replace` directives (`plugins/session-tmux` module)
- [x] 8.2 Create `plugins/session-tmux/internal/session/tmux.go`: `Tmux` struct implementing `pluginv1.SessionServer`; takes a `tmux` binary path (for testing injection); shells out to tmux for all operations; no global state (`plugins/session-tmux` module)
- [x] 8.3 Implement `OpenWorkspace`: compute socket path from story name, start tmux server if socket absent, create one session per worktree_path entry, attach via `tmux attach-session`; return `Workspace` (`plugins/session-tmux` module)
- [x] 8.4 Implement `CloseWorkspace`: run `tmux -S <socket> kill-server`, remove socket file; idempotent if socket absent (`plugins/session-tmux` module)
- [x] 8.5 Implement `ListWorkspaces`: scan `$XDG_RUNTIME_DIR/swm/tmux/`, probe each socket, stream live `Workspace` messages (`plugins/session-tmux` module)
- [x] 8.6 Implement `OpenPaneGroup`: create tmux session named after last project segment in the story's socket; apply `pane_group_command` config or default `$EDITOR + $SHELL` layout; idempotent (`plugins/session-tmux` module)
- [x] 8.7 Implement `SwitchTo`: use `switch-client` if `$TMUX` is set, else `attach-session` (`plugins/session-tmux` module)
- [x] 8.8 Implement `IsInsideWorkspace`: check `$TMUX` env var and match socket path prefix against swm's socket dir (`plugins/session-tmux` module)
- [x] 8.9 Implement `CurrentContext`: parse socket path from `$TMUX` to derive story name; query active session name from `tmux display-message`; return `CurrentContextResponse` (`plugins/session-tmux` module)
- [x] 8.10 Implement `Info`: return `SessionInfo{plugin_info: {name: "tmux", version: buildVersion}}` (`plugins/session-tmux` module)
- [x] 8.11 Update `plugins/session-tmux/main.go`: call `session.Serve(&Tmux{})` instead of stub (`plugins/session-tmux` module)
- [x] 8.12 Write tests in `plugins/session-tmux/internal/session/tmux_test.go` using a fake `tmux` script (shell script in testdata that records invocations); cover OpenWorkspace, CloseWorkspace, ListWorkspaces, IsInsideWorkspace, CurrentContext (`plugins/session-tmux` module)
- [x] 8.13 Run `task session-tmux:test` → confirm exits 0

## 9. CLI Commands (cmd/swm)

- [x] 9.1 Create `cmd/swm/internal/cli/root.go`: `NewRootCmd(cfg *config.Config, mgr *pluginmgr.Manager, store story.Store) *cobra.Command`; wire config, manager, and store via constructor injection; no global state (`cmd/swm` module)
- [x] 9.2 Create `cmd/swm/internal/cli/story/create.go`: `NewCreateCmd(store story.Store) *cobra.Command` implementing `swm story create <name> [--branch <branch>]` per spec (`cmd/swm` module)
- [x] 9.3 Create `cmd/swm/internal/cli/story/remove.go`: `NewRemoveCmd(store story.Store, mgr *pluginmgr.Manager, layout *layout.Resolver) *cobra.Command` implementing `swm story remove <name> [--force]`; best-effort cleanup with error summary (`cmd/swm` module)
- [x] 9.4 Create `cmd/swm/internal/cli/clone.go`: `NewCloneCmd(mgr *pluginmgr.Manager, layout *layout.Resolver) *cobra.Command` implementing `swm clone <url>` per spec (`cmd/swm` module)
- [x] 9.5 Create `cmd/swm/internal/cli/workspace/open.go`: `NewOpenCmd(store story.Store, mgr *pluginmgr.Manager, layout *layout.Resolver) *cobra.Command` implementing `swm workspace open [--story <name>]` per spec (`cmd/swm` module)
- [x] 9.6 Update `cmd/swm/main.go`: initialize config, hostsvc server, plugin manager, story store, layout resolver; pass to `NewRootCmd`; call `manager.Close()` on exit (`cmd/swm` module)
- [x] 9.7 Write unit tests for each CLI command using `cobra` `ExecuteC` with stub implementations of `Store` and `Manager` interfaces; cover success paths and error cases from the spec scenarios (`cmd/swm` module)

## 10. Integration Tests and Final Verification

- [x] 10.1 Create `cmd/swm/tests/integration/` package; write a `TestMain` that compiles `plugins/vcs-git` and `plugins/session-tmux` binaries to a temp dir and adds that dir to PATH (`cmd/swm` module)
- [x] 10.2 Write `TestCloneAndStoryCreate`: real git repo on local filesystem, `swm clone file:///path/to/repo`, then `swm story create feat-x`; verify canonical path and story JSON on disk (`cmd/swm` module)
- [x] 10.3 Write `TestStoryRemove`: create story + worktree, `swm story remove --force`; verify worktree and story JSON are removed (`cmd/swm` module)
- [x] 10.4 Write `TestWorkspaceOpen` with a mock tmux binary (records invocations, always exits 0); verify `OpenWorkspace` is called with correct story name and worktree paths (`cmd/swm` module)
- [x] 10.5 Run `task fmt` → confirm exits 0
- [x] 10.6 Run `task lint` → confirm exits 0
- [x] 10.7 Run `task test` → confirm exits 0
