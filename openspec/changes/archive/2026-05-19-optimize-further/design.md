## Context

`Manager.Get` holds a global `sync.Mutex` for the full duration of plugin
launch (exec + gRPC handshake, ~300–400 ms per plugin). Two concurrent callers
of `Get` therefore serialize completely. As a result, even if we pre-warm
multiple plugins before the command body runs, they launch one-at-a-time and
save nothing. True parallel startup requires per-capability locking.

Current capability usage by command:

| Command          | Capabilities used         |
|------------------|---------------------------|
| workspace open   | picker (optional), session, vcs |
| workspace list   | none                      |
| clone            | vcs                       |
| story remove     | vcs, session (optional)   |
| pr list/create   | forge (via GetForge)      |

## Goals / Non-Goals

**Goals:**
- Eliminate serial plugin startup for commands that use multiple capabilities.
- Keep the lazy `Get` path fully functional for capabilities not declared upfront.
- Expose `Warm(ctx, capabilities...)` for commands to call in `PreRunE`.

**Non-Goals:**
- Warming forge plugins (GetForge has its own separate loading path; out of scope).
- Changing plugin discovery, binary resolution, or the dep-graph validation.
- Retrying failed plugin launches (consistent with current behavior intent).

## Decisions

### Decision 1: Per-capability `sync.Once` replaces global mutex

Replace `sync.Mutex + map[string]*entry` with `sync.Map` of `*launchOnce`:

```go
type launchOnce struct {
    once sync.Once
    raw  any
    err  error
}
```

`LoadOrStore` atomically ensures only one `*launchOnce` per capability key.
`once.Do` ensures only one goroutine actually executes the launch. All other
goroutines block on `once.Do` then read the shared result. This allows `Warm`
to fan out `Get` calls across goroutines with true concurrency.

**Alternative considered**: keep the global mutex and add a per-capability
`sync.Cond`. Rejected: more complex, same result.

**Trade-off**: failed launches are now cached (once.Do never reruns). This is
acceptable — if a binary is missing at startup it will still be missing on
retry, and deterministic errors are preferable to non-deterministic retries.
Current behavior (retry on failure) was an accidental side-effect, not a design
goal.

### Decision 2: `Warm` fans out goroutines, collects errors with `errgroup`

```go
func (m *Manager) Warm(ctx context.Context, capabilities ...string) error {
    g, ctx := errgroup.WithContext(ctx)
    for _, cap := range capabilities {
        g.Go(func() error { _, err := m.Get(ctx, cap); return err })
    }
    return g.Wait()
}
```

Using `errgroup` cancels the context on first error, propagating cancellation
to in-flight launches. Commands treat a `Warm` error as fatal in `PreRunE`.

**Alternative**: ignore errors in `Warm` and let the command body fail on `Get`.
Rejected: silent pre-warm failures make debugging harder; fail-fast at `PreRunE`
gives a clean error before any user interaction.

### Decision 3: `Warm` added to the `PluginManager` interface

`PluginManager` (defined in `cmd/swm/internal/cli/root.go`) gains:
```go
Warm(ctx context.Context, capabilities ...string) error
```

Commands access `mgr` through this interface, so `Warm` must be part of it.

**Alternative**: define a separate `PluginWarmer` interface per command.
Rejected: unnecessary fragmentation — all commands already receive the same
`mgr` argument.

### Decision 4: Commands declare capabilities via `PreRunE`

Each command that benefits adds its own `PreRunE`:

```go
PreRunE: func(cmd *cobra.Command, _ []string) error {
    return mgr.Warm(cmd.Context(), "picker", "session", "vcs")
},
```

`PreRunE` runs after the root's `PersistentPreRunE` (log-level setup), so
the logger is configured before any plugin is launched.

Commands covered:
- `workspace open`: `["picker", "session", "vcs"]`
- `clone`: `["vcs"]`
- `story remove`: `["vcs", "session"]`

`workspace list` and `pr` commands are unchanged (no `PreRunE`).

**Alternative**: root-level `PersistentPreRunE` inspects the active command and
warms a registered map. Rejected: couples root to every subcommand's
capability requirements; harder to test.

## Risks / Trade-offs

- **Error caching**: A transiently missing binary (e.g. not yet on PATH during
  `PreRunE`) will cache the error and fail subsequent `Get` calls for the
  lifetime of the process. In practice this cannot happen — binaries on PATH
  don't appear mid-run. Acceptable.

- **`sync.Map` overhead**: `sync.Map` is heavier than a plain map+mutex for
  write-heavy workloads. Plugin launch happens at most once per capability per
  process; the map is effectively read-only after warmup. Overhead is negligible.

- **`Close` races**: `Close` must drain `sync.Map` safely. Use `Range` +
  `Delete` instead of the old `m.launched = make(map[string]*entry)` reset.

## Open Questions

_(none — scope is clear)_
