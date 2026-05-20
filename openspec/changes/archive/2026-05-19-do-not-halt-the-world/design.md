## Context

`Manager.Warm()` currently blocks the caller until every requested plugin has started
(via `wg.Wait()`). In `PreRunE`, this means the command's `RunE` cannot begin until
all plugins are up, defeating the purpose of early warming. Plugin startup can take
hundreds of milliseconds for the gRPC handshake.

The existing `launchOnce` / `sync.Once` machinery already provides the right primitive:
`once.Do` blocks all subsequent callers until the first invocation completes. The only
missing piece is decoupling `Warm`'s goroutines from the caller's wait.

## Goals / Non-Goals

**Goals:**
- `Warm` returns immediately after firing background goroutines.
- `Get` remains the natural synchronization point — blocks only if the plugin is still
  booting, returns immediately if it's ready.
- `PreRunE` call sites simplify: one `Warm` call, no `sync.WaitGroup`, no error check.
- Context lifetime is correct: background goroutines outlive `PreRunE`.

**Non-Goals:**
- Changing the plugin gRPC protocol or any proto definitions.
- Adding UI feedback (spinner, log line) during background startup.
- Removing `Warm` call sites — they remain as early-start hints.

## Decisions

### 1. Warm fires goroutines and returns nil

**Decision:** `Warm` iterates capabilities, calls `m.launched.LoadOrStore` to
get-or-create the `launchOnce` entry, then fires a goroutine that calls
`lo.once.Do(launch)`. It returns `nil` immediately.

**Why:** Errors from plugin startup are already cached in `launchOnce.err` and
returned by every subsequent `Get`. There is no value in returning them from
`Warm` — the caller can't act on them without blocking anyway.

**Alternative considered:** Keep `Warm` blocking but add a timeout. Rejected —
any timeout is arbitrary and the problem recurs with slow machines.

### 2. Background goroutine uses a detached context

**Decision:** `Warm` wraps the caller's context with `context.WithoutCancel` before
passing it to the goroutine. This strips deadline/cancellation while preserving values
(logger, trace IDs).

**Why:** `PreRunE`'s context may be cancelled or have a short deadline. The background
goroutine must survive until `Get` is called from `RunE`. `context.WithoutCancel` is
the minimal change — background context would lose values.

**Why not `context.Background()`:** Loses logger and any other values stored in the
context chain.

### 3. Get is unchanged

**Decision:** `Get` remains exactly as-is. `sync.Once.Do` blocks callers until the
first invocation (the warm goroutine's `once.Do`) completes, then all subsequent
callers return the cached result.

**Why:** The synchronization is already correct. The warm goroutine "owns" the
`once.Do` if it starts first; `Get` will block until it finishes. If `Get` starts
first (warm never called or lost the race), it runs launch itself.

### 4. PreRunE call sites consolidate

**Decision:** All `PreRunE` hooks that call `Warm` simplify to a single `mgr.Warm(ctx, caps...)`
with no error check and no `sync.WaitGroup`. Picker is no longer special-cased.

```go
// Before
var wg sync.WaitGroup
wg.Go(func() { _ = mgr.Warm(cmd.Context(), "picker") })
err := mgr.Warm(cmd.Context(), "session", "vcs")
wg.Wait()
return err

// After
mgr.Warm(cmd.Context(), "picker", "session", "vcs")
return nil
```

**Why:** Warm errors now surface in RunE when `Get` is called. The caller-side
`WaitGroup` was compensating for Warm being synchronous; it is no longer needed.

## Risks / Trade-offs

- **[Risk] Error surfacing moves later** → Mitigation: errors appear at first `Get`
  call in `RunE`, which is still early and with the same message. The only behavioral
  difference is that a missing plugin binary is reported slightly later.

- **[Risk] Context lifetime confusion** → Mitigation: `context.WithoutCancel` is
  explicit and documented at the call site. The pattern is standard for "fire and
  forget with value inheritance."

- **[Risk] Background goroutine leak if process exits before Get** → Mitigation:
  `Manager.Close` kills plugin processes; if the process exits before `Get` is called,
  the goroutine will return quickly (launch fails or the OS reclaims the child).

## Open Questions

None.
