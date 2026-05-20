## MODIFIED Requirements

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
