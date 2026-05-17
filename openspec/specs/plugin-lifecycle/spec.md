### Requirement: Plugin discovery
The plugin manager SHALL discover plugin binaries in the following priority order, stopping at the first match per capability: (1) explicit paths from `config.toml [plugins.paths]`, (2) `$XDG_DATA_HOME/swm/plugins/<name>/swm-plugin-<capability>-<name>`, (3) `$PATH` lookup for `swm-plugin-<capability>-<name>`. The binary naming convention MUST be `swm-plugin-<capability>-<name>`.

#### Scenario: PATH discovery
- **WHEN** no explicit path or XDG path is configured for capability `vcs` and `swm-plugin-vcs-git` is in `$PATH`
- **THEN** the manager discovers `swm-plugin-vcs-git` as the vcs plugin

#### Scenario: Explicit config overrides PATH
- **WHEN** `config.toml` sets an explicit path for `vcs` and `swm-plugin-vcs-git` is also in `$PATH`
- **THEN** the manager uses the explicitly configured binary, not the PATH one

#### Scenario: Missing required plugin
- **WHEN** `config.toml` specifies `vcs = "git"` and no `swm-plugin-vcs-git` is found in any search location
- **THEN** `Manager.Get("vcs")` returns an error describing which capability binary was not found

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

### Requirement: Lazy launch
The plugin manager SHALL NOT launch any plugin binary at `Manager` creation time. Plugins SHALL be launched on the first call to `Manager.Get(capability)` for that capability. Subsequent calls to `Manager.Get` for the same capability SHALL return the already-launched client without relaunching.

#### Scenario: First Get triggers launch
- **WHEN** `Manager.Get("vcs")` is called for the first time
- **THEN** exactly one `swm-plugin-vcs-git` process is spawned

#### Scenario: Second Get reuses client
- **WHEN** `Manager.Get("vcs")` is called twice
- **THEN** only one plugin process exists; the second call returns the cached client

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
