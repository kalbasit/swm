### Requirement: Plugin discovery
The plugin manager SHALL discover plugin binaries in the following priority order,
stopping at the first match per capability:
(0) each directory listed in `$SWM_PLUGIN_PATH` (colon-separated, evaluated
left-to-right) — non-existent or non-directory entries MUST be silently skipped,
(1) explicit paths from `config.toml [plugins.paths]`,
(2) `$XDG_DATA_HOME/swm/plugins/<name>/swm-plugin-<capability>-<name>`,
(3) `$PATH` lookup for `swm-plugin-<capability>-<name>`.
The binary naming convention MUST be `swm-plugin-<capability>-<name>`.

#### Scenario: PATH discovery
- **WHEN** no explicit path or XDG path is configured for capability `vcs` and `swm-plugin-vcs-git` is in `$PATH`
- **THEN** the manager discovers `swm-plugin-vcs-git` as the vcs plugin

#### Scenario: Explicit config overrides PATH
- **WHEN** `config.toml` sets an explicit path for `vcs` and `swm-plugin-vcs-git` is also in `$PATH`
- **THEN** the manager uses the explicitly configured binary, not the PATH one

#### Scenario: Missing required plugin
- **WHEN** `config.toml` specifies `vcs = "git"` and no `swm-plugin-vcs-git` is found in any search location
- **THEN** `Manager.Get("vcs")` returns an error describing which capability binary was not found

#### Scenario: SWM_PLUGIN_PATH takes precedence over all other search locations
- **WHEN** `SWM_PLUGIN_PATH=/dev/bin`, `swm-plugin-vcs-git` exists in `/dev/bin`, and `swm-plugin-vcs-git` is also discoverable via `$PATH`
- **THEN** the manager uses `/dev/bin/swm-plugin-vcs-git`

#### Scenario: SWM_PLUGIN_PATH takes precedence over explicit config paths
- **WHEN** `SWM_PLUGIN_PATH=/dev/bin`, `swm-plugin-vcs-git` exists in `/dev/bin`, and `config.toml` sets an explicit path for `vcs` pointing elsewhere
- **THEN** the manager uses `/dev/bin/swm-plugin-vcs-git`

#### Scenario: SWM_PLUGIN_PATH colon-separated list searched left-to-right
- **WHEN** `SWM_PLUGIN_PATH=/dir1:/dir2` and `swm-plugin-vcs-git` exists only in `/dir2`
- **THEN** the manager discovers `/dir2/swm-plugin-vcs-git`

#### Scenario: Non-existent SWM_PLUGIN_PATH entries are silently skipped
- **WHEN** `SWM_PLUGIN_PATH=/nonexistent:/dir2` and `swm-plugin-vcs-git` exists in `/dir2`
- **THEN** `/nonexistent` is silently skipped and the manager discovers `/dir2/swm-plugin-vcs-git`

#### Scenario: Unset SWM_PLUGIN_PATH leaves discovery unchanged
- **WHEN** `SWM_PLUGIN_PATH` is not set and `swm-plugin-vcs-git` is in `$PATH`
- **THEN** the manager discovers `swm-plugin-vcs-git` via PATH as before

### Requirement: Plugin launch via go-plugin
The plugin manager SHALL launch plugins using go-plugin's gRPC client. The magic cookie key MUST be `SWM_PLUGIN_MAGIC_COOKIE` and the value MUST be `swm-plugin-v1` (from `sdk/go/handshake`). Launched plugin processes MUST be terminated when `Manager.Close()` is called.

#### Scenario: Successful launch and handshake
- **WHEN** `Manager.Get("vcs")` is called and `swm-plugin-vcs-git` is discoverable
- **THEN** the binary is launched, go-plugin handshake completes, and a typed `VCSClient` is returned

#### Scenario: Wrong magic cookie
- **WHEN** a binary exists with the correct name but does not output the expected go-plugin handshake
- **THEN** `Manager.Get` returns an error indicating handshake failure

#### Scenario: Process cleanup on close
- **WHEN** `Manager.Close()` is called after one or more plugins are launched
- **THEN** all launched plugin processes are terminated (no zombie processes)

### Requirement: Parallel eager plugin warm-up
The plugin manager SHALL expose a `Warm(ctx context.Context, capabilities ...string) error`
method that starts all listed plugins in background goroutines and returns immediately with nil.
Plugin startup errors are NOT returned by `Warm`; they are surfaced by the first `Get` call
for that capability. Plugins already launched are reused without re-launching.
The context passed to `Warm` SHALL be stripped of cancellation so background goroutines
outlive the caller's context.

#### Scenario: Two capabilities warm concurrently
- **WHEN** `Warm(ctx, "vcs", "session")` is called
- **THEN** both plugins begin starting in background goroutines
- **THEN** `Warm` returns nil immediately without waiting for either to complete

#### Scenario: Warm with already-launched capability
- **WHEN** `Get(ctx, "vcs")` has already been called and `Warm(ctx, "vcs")` is called
- **THEN** no new process is spawned; `Warm` returns nil

#### Scenario: Warm does not propagate launch errors
- **WHEN** `Warm(ctx, "picker")` is called and the picker binary is missing
- **THEN** `Warm` returns nil immediately
- **WHEN** `Get(ctx, "picker")` is subsequently called
- **THEN** `Get` returns the launch error describing the missing binary

### Requirement: Get blocks if background warm is in progress
`Get` SHALL block until any background launch started by `Warm` for the requested
capability completes, then return the cached result. This preserves the invariant
that `Get` always returns either a valid client or a deterministic cached error.

#### Scenario: Get waits for in-progress background warm
- **WHEN** `Warm(ctx, "vcs")` fires a background goroutine that has not yet completed
- **WHEN** `Get(ctx, "vcs")` is called concurrently
- **THEN** `Get` blocks until the background launch finishes
- **THEN** `Get` returns the launched client (or cached error if launch failed)

#### Scenario: Get after warm completes
- **WHEN** `Warm(ctx, "vcs")` background goroutine has already finished
- **WHEN** `Get(ctx, "vcs")` is called
- **THEN** `Get` returns immediately with the cached client

### Requirement: Commands declare capabilities for pre-warming
Commands that need specific plugins SHALL call `Warm` in `PreRunE` to initiate background
startup early. `PreRunE` SHALL return immediately after calling `Warm` without blocking on
plugin readiness. Startup errors surface in `RunE` when the plugin is first used via `Get`.

#### Scenario: workspace open initiates warm without blocking
- **WHEN** `workspace open` PreRunE runs
- **THEN** `Warm` is called with all required capabilities (session, vcs, picker)
- **THEN** PreRunE returns immediately without waiting for any plugin to be ready

#### Scenario: Commands with no static capabilities do not warm
- **WHEN** a command has no known plugin dependencies
- **THEN** its `PreRunE` does not call `Warm`

### Requirement: Lazy launch
The plugin manager SHALL NOT launch any plugin binary at `Manager` creation
time. Plugins SHALL be launched on the first call to `Manager.Get(capability)`
or `Manager.Warm(ctx, capability)` for that capability. Subsequent calls to
`Manager.Get` or `Manager.Warm` for the same capability SHALL return the
already-launched client without relaunching.

A failed launch SHALL be cached: subsequent calls for the same capability
return the same error without retrying. (Prior behavior retried on failure as
an accidental side-effect of the map-not-populated-on-error pattern; this
change makes the behavior explicit and deterministic.)

#### Scenario: First Get triggers launch
- **WHEN** `Manager.Get("vcs")` is called for the first time
- **THEN** exactly one `swm-plugin-vcs-git` process is spawned

#### Scenario: Second Get reuses client
- **WHEN** `Manager.Get("vcs")` is called twice
- **THEN** only one plugin process exists; the second call returns the cached client

#### Scenario: Failed launch is not retried
- **WHEN** `Manager.Get("picker")` fails because the binary is missing
- **THEN** a second call to `Manager.Get("picker")` returns the same error without spawning a new process

### Requirement: Capability dep-graph validation
The plugin manager SHALL call `Info()` on each launched plugin to read its `PluginInfo`. After loading all configured plugins, it SHALL validate that every required capability declared in `requires` is satisfied by another configured plugin. An unsatisfied required dep MUST cause an error: "plugin <name> requires capability <cap>, but no <cap> plugin is configured."

#### Scenario: Satisfied required dep
- **WHEN** `session-tmux` declares `requires: [vcs]` and a vcs plugin is configured
- **THEN** validation passes and the manager proceeds normally

#### Scenario: Unsatisfied required dep
- **WHEN** `session-tmux` declares `requires: [vcs]` but no vcs plugin is configured
- **THEN** `Manager.Get("session")` returns an error describing the unsatisfied dependency

#### Scenario: Missing optional dep does not error
- **WHEN** a plugin declares `optional: ["forge"]` and no forge plugin is configured
- **THEN** validation passes; the plugin is launched without access to the forge capability

### Requirement: Host service callback
When launching each plugin, the plugin manager SHALL pass the address of the in-process Host gRPC server via the `SWM_HOST_SOCKET` environment variable. The Host server SHALL implement `GetConfig`, `GetCodeRoot`, `ListProjects`, and `Log` RPCs. `GetConfig` SHALL return only the TOML subtable scoped to the plugin's name (`[plugins.config.<name>]`).

#### Scenario: Plugin reads its own config
- **WHEN** a plugin calls `host.GetConfig({})` and `config.toml` contains `[plugins.config.vcs-git] foo = "bar"`
- **THEN** the response contains TOML bytes for `foo = "bar"` only (not other plugin configs)

#### Scenario: Plugin logs via host
- **WHEN** a plugin calls `host.Log({level: "info", message: "hello"})`
- **THEN** the message is written to the host's structured logger at the specified level

### Requirement: Plugin manager logger level propagation
The plugin manager SHALL configure go-plugin's internal `hclog` logger to the same level as the host's active log level (`--log-level` flag). When no level is explicitly configured, the hclog logger SHALL default to `warn`. At no point SHALL the hclog logger use a level lower than the host's configured level (i.e., it MUST NOT emit DEBUG or TRACE lines when the host is at `warn` or higher).

#### Scenario: Default log level suppresses DEBUG output
- **WHEN** `swm` is invoked without `--log-level` (default is `warn`)
- **THEN** no `[DEBUG]` or `[TRACE]` lines from go-plugin appear on stderr during plugin launch

#### Scenario: Debug log level shows go-plugin internal lines
- **WHEN** `swm` is invoked with `--log-level debug`
- **THEN** go-plugin `[DEBUG]` lines for plugin startup, handshake, and RPC address appear on stderr

#### Scenario: Warn level preserves plugin error visibility
- **WHEN** `swm` is invoked without `--log-level` and a plugin process crashes
- **THEN** go-plugin `[WARN]` or `[ERROR]` lines for the crash appear on stderr

### Requirement: Plugin-internal environment variable contract

The plugin manager defines a fixed set of environment variables that are classified as plugin-internal. These variables MUST be present in every plugin subprocess environment and MUST NOT appear in any process that is not a designated plugin subprocess.

Plugin-internal variables:

| Variable | Purpose |
|---|---|
| `SWM_HOST_SOCKET` | gRPC socket address for plugin-to-host callbacks |
| `SWM_LOG_LEVEL` | host log level propagated to plugin logging |
| `SWM_PLUGIN_MAGIC_COOKIE` | go-plugin handshake token |

#### Scenario: Plugin subprocess receives all plugin-internal vars
- **WHEN** the plugin manager launches any plugin subprocess
- **THEN** `SWM_HOST_SOCKET`, `SWM_LOG_LEVEL`, and `SWM_PLUGIN_MAGIC_COOKIE` are all present in that process's environment

#### Scenario: Plugin-internal vars are not required in non-plugin processes
- **WHEN** the session plugin launches a terminal multiplexer process
- **THEN** `SWM_HOST_SOCKET`, `SWM_LOG_LEVEL`, and `SWM_PLUGIN_MAGIC_COOKIE` are absent from that process's environment

#### Scenario: Plugin-internal vars are not required in hook processes
- **WHEN** the hook executor launches a hook binary
- **THEN** `SWM_HOST_SOCKET`, `SWM_LOG_LEVEL`, and `SWM_PLUGIN_MAGIC_COOKIE` are absent from that process's environment

### Requirement: Plugin manager interface exposes Close
The `pluginManager` interface consumed by any command that may call `exec`
SHALL include a `Close() error` method. This ensures the command can
explicitly terminate plugin subprocesses before replacing the process image.

#### Scenario: Close method present on pluginManager interface
- **WHEN** `workspace open` is compiled
- **THEN** the `pluginManager` interface in the `workspace` package includes `Close() error`

#### Scenario: Concrete manager satisfies extended interface
- **WHEN** `*pluginmgr.Manager` is assigned to a `pluginManager` variable
- **THEN** the assignment compiles without error (i.e. `Manager` already implements `Close()`)

### Requirement: Plugins are terminated before exec replaces the process
When `workspace open` is about to call `exec` to replace the swm process
with a terminal multiplexer client, it SHALL call `mgr.Close()` first.
No plugin subprocess SHALL outlive the `swm workspace open` invocation.

#### Scenario: No plugin processes remain after successful workspace open
- **WHEN** `swm workspace open` completes successfully and exec replaces the process
- **THEN** all plugin subprocesses spawned during that invocation have been terminated

#### Scenario: Close is called before exec
- **WHEN** `SwitchTo` returns an exec argv and `execFn` is about to be called
- **THEN** `mgr.Close()` is called before `execFn` is invoked

#### Scenario: Close error does not prevent exec
- **WHEN** `mgr.Close()` returns a non-nil error
- **THEN** the error is logged but does not prevent `execFn` from being called
- **THEN** the workspace open flow continues to exec

#### Scenario: Defer still covers non-exec error paths
- **WHEN** `workspace open` returns an error before reaching the exec path
- **THEN** the deferred `mgr.Close()` in `main.go` still terminates plugin subprocesses
