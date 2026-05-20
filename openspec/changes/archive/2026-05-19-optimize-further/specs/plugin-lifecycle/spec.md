## ADDED Requirements

### Requirement: Parallel eager plugin warm-up
The plugin manager SHALL expose a `Warm(ctx context.Context, capabilities ...string) error`
method that starts all listed plugins concurrently. `Warm` SHALL return the
first error encountered; on error the context passed to remaining in-flight
launches is cancelled. Plugins that have already been launched (by a prior
`Warm` or `Get` call) SHALL be reused without re-launching.

#### Scenario: Two capabilities warm concurrently
- **WHEN** `Warm(ctx, "picker", "session")` is called and neither plugin is running
- **THEN** both plugin processes are started concurrently (not one after the other)

#### Scenario: Warm with already-launched capability
- **WHEN** `Get(ctx, "vcs")` has already been called and `Warm(ctx, "vcs")` is called
- **THEN** no new process is spawned; `Warm` returns nil

#### Scenario: Warm propagates launch errors
- **WHEN** `Warm(ctx, "picker", "session")` is called and the picker binary is missing
- **THEN** `Warm` returns a non-nil error describing the missing binary

### Requirement: Commands declare capabilities for pre-warming
Commands that use a fixed set of capabilities SHALL declare those capabilities
by calling `Warm` in their `PreRunE` hook. `PreRunE` runs after root-level
log-level setup but before the command body, ensuring plugins are ready before
any user interaction begins.

Capability declarations by command:

| Command        | Pre-warmed capabilities     |
|----------------|-----------------------------|
| workspace open | session, vcs                |
| clone          | vcs                         |
| story remove   | vcs, session                |

Note: `picker` is optional for `workspace open` (errors are ignored in the command
body), so it is NOT pre-warmed — a missing picker binary would otherwise cause
`PreRunE` to fail before the command can fall back gracefully to non-interactive mode.

Commands with no statically-known capabilities (workspace list, pr list, pr
create) SHALL NOT call `Warm`.

#### Scenario: workspace open warms before picker is invoked
- **WHEN** `swm workspace open <story>` is run
- **THEN** picker and session plugins are already running before the fuzzy picker UI is displayed

#### Scenario: Commands with no static capabilities do not warm
- **WHEN** `swm workspace list` is run
- **THEN** no plugin processes are spawned

## MODIFIED Requirements

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
