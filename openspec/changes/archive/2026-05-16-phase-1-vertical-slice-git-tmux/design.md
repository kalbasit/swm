## Context

Phase 0 delivered buildable proto definitions, SDK stubs, and plugin binary skeletons. Nothing does anything yet. Phase 1 wires in real behavior across four modules — `cmd/swm`, `sdk/go`, `plugins/vcs-git`, `plugins/session-tmux` — to produce a vertical slice a user can actually run.

Key constraints inherited from the TDD:

- **Host owns all paths.** Plugins return `ProjectID(host, segments[])`, host composes every on-disk path. No plugin chooses layout.
- **go-plugin over gRPC.** All plugin communication uses HashiCorp go-plugin with the existing handshake constants from `sdk/go/handshake`.
- **TOML config** via `pelletier/go-toml/v2`. No viper, no YAML.
- **flock on story writes.** Concurrent `swm` invocations must not corrupt story JSON files.
- **XDG layout.** Stories at `$XDG_DATA_HOME/swm/stories/<name>.json`, config at `$XDG_CONFIG_HOME/swm/config.toml`.
- **Conventional Commits.** Interfaces defined in consumer packages, not shared `ifaces/`.

## Goals / Non-Goals

**Goals:**
- Story store with JSON persistence, flock, and full CRUD
- Config loader parsing `config.toml`, returning typed structs
- Plugin manager: discover → launch → handshake → dep-graph → capability registry
- Host service implementations (GetConfig, GetCodeRoot, ListProjects, Log)
- SDK: real go-plugin GRPCPlugin registration for session and vcs capabilities
- vcs-git plugin: all five RPCs needed by Phase 1 flows (ParseRemoteURL, Clone, CreateWorktree, RemoveWorktree, DetectProjectAtPath)
- session-tmux plugin: all seven RPCs (OpenWorkspace, CloseWorkspace, ListWorkspaces, OpenPaneGroup, SwitchTo, IsInsideWorkspace, CurrentContext)
- Four CLI commands wiring the above: `swm story create`, `swm story remove`, `swm clone`, `swm workspace open`
- Integration tests using real git and real filesystem; session-tmux tests mock the tmux binary

**Non-Goals:**
- Picker (Phase 2), forge/hooks (Phase 3), plugin install (Phase 4)
- `CallCapability` routing (no plugin-to-plugin calls needed in Phase 1 flows)
- Windows support (Unix sockets only)
- Plugin binary sandboxing

## Decisions

### D1: SDK transport layer — GRPCPlugin per capability

Each `sdk/go/<cap>/plugin.go` defines a `GRPCPlugin` that implements `go-plugin`'s `GRPCPlugin` interface, wrapping the generated proto server/client. `Serve(impl Plugin)` registers this plugin with go-plugin and calls `plugin.Serve()`. The host side gets a `NewClient()` constructor returning a typed capability client.

Alternatives considered:
- A single shared `sdk/go/transport` package — rejected, would require every plugin to import all capability types; per-capability packages keep imports minimal.
- Direct gRPC without go-plugin — rejected, we need the process-lifecycle management go-plugin provides (magic cookie, signal handling, socket cleanup).

### D2: Config loading — simple struct with defaults

`cmd/swm/internal/config/` provides a `Load(path string) (*Config, error)` function using `pelletier/go-toml/v2`. The `Config` struct has explicit default values applied before unmarshaling (code_root defaults to `~/code`, default_story to `_default`). No global config variable; callers pass `*Config` explicitly.

Per-plugin config sections are unmarshaled as `map[string]toml.Primitive` and decoded lazily by each plugin via `host.GetConfig()`.

Alternative: viper — rejected per TDD, YAML/TOML ambiguity, and no global state policy.

### D3: Story store — interface + JSON file implementation

`cmd/swm/internal/core/story/` defines:
```
type Store interface {
    Create(ctx, name, branchName string) (*Story, error)
    Get(ctx, name string) (*Story, error)
    List(ctx context.Context) ([]*Story, error)
    Delete(ctx, name string) error
    Update(ctx context.Context, s *Story) error
}
```
`JSONStore` implements it: each story is `$XDG_DATA_HOME/swm/stories/<name>.json`. Writes use `gofrs/flock` on the file path before writing, released via `defer`. Reads do not lock (readers are safe; writes are protected).

Alternative: SQLite — rejected, adds binary dep, overkill for a small set of stories per user.

### D4: Plugin manager — lazy launch, per-invocation pool

`cmd/swm/internal/pluginmgr/Manager` holds a registry of discovered plugin paths. On first `Get(capability)` call it launches the binary, performs go-plugin handshake, calls `Info()`, populates the capability registry. Subsequent calls return the cached client. On `Close()`, all plugin processes are terminated. The `Manager` is created once per CLI invocation in `main()` and passed down via explicit dependency injection.

Dep-graph validation runs after all configured plugins are discovered (lazy discovery on first `Get` means we validate on first actual use, not at startup — this keeps `--version` and `--help` fast).

Alternative: eager startup — rejected, latency cost on every invocation even when no plugins are needed.

### D5: vcs-git — shell out to system git

The vcs-git plugin shells out to `git` via `os/exec` rather than using `go-git`. Reasons: (1) go-git has incomplete SSH agent support and credential helper integration; (2) system git handles all edge cases users actually encounter (shallow clones, LFS, custom transports); (3) binary size stays small.

All `git` invocations check exit code and capture stderr for error messages returned as gRPC status errors.

Alternative: go-git library — rejected (see above).

### D6: session-tmux — shell out to tmux

Same rationale as D5. The plugin shells out to `tmux` with structured flags. Each workspace maps to a dedicated tmux server socket at `$XDG_RUNTIME_DIR/swm/tmux/<story-name>.sock`. Each pane group maps to a tmux session (named after the project's last path segment) within that socket. This preserves the v1 tmux model exactly.

For `OpenWorkspace`, the plugin creates the socket's server if it doesn't exist (`tmux -S <socket> new-session -d -s <pane-group> -c <worktree-path>`), then attaches via `tmux -S <socket> attach-session`.

### D7: Host services — gRPC server on a Unix socket

When the plugin manager launches a plugin, it also starts an in-process gRPC server for the Host service. The server's Unix socket path is passed to the plugin via env var `SWM_HOST_SOCKET`. Plugin implementations receive this address in their env and dial it to obtain a `HostClient`. This allows plugin → host callbacks (GetConfig, GetCodeRoot, ListProjects, Log) without a second go-plugin subprocess.

The Host server is started once, shared across all plugins, and shut down after all plugins are closed.

## Risks / Trade-offs

- **tmux socket cleanup on crash**: If swm crashes without calling Close(), tmux sockets in `$XDG_RUNTIME_DIR/swm/tmux/` persist. Mitigation: `OpenWorkspace` checks for existing sockets and reuses them rather than erroring; cleanup is deferred to `swm story remove`.

- **go-git vs system git divergence for tests**: Integration tests that require git need git installed. Mitigation: tests run in the Nix devshell where git is guaranteed. CI uses the same Nix environment.

- **XDG_DATA_HOME undefined in some environments**: `adrg/xdg` handles the fallback to `~/.local/share` per the XDG spec. No special-casing needed.

- **Plugin binary not in PATH during tests**: Integration tests for the CLI commands need the compiled plugin binaries available. Mitigation: test helpers compile the plugin binaries to a temp dir and inject that dir into PATH for the test process.

## Open Questions

- Should `swm workspace open` without a current story use the `_default` story or error? (TDD §7.4 says "resolve story from flag, $SWM_STORY, or default" — treating missing as `_default` is consistent with v1.)
- Should `swm clone` automatically create a story and worktree, or just clone to canonical? (TDD §7.2 says "not attached to any story yet" — clone only, no story.)
