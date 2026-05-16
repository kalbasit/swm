## ADDED Requirements

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
