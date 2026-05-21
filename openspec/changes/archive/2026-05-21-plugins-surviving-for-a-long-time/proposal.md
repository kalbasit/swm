## Why

When `swm workspace open` completes, it calls `syscall.Exec` to replace the
swm process with `tmux attach-session`. Because `exec` replaces the process
image, all deferred calls — including `mgr.Close()` — are skipped, leaving
plugin subprocesses alive as orphaned children of the tmux client. Each
subsequent `swm workspace open` invocation spawns a fresh set of plugins,
accumulating orphans across days of use.

## What Changes

- Add `Close() error` to the `pluginManager` interface consumed by
  `cmd/swm/internal/cli/workspace/open.go`.
- Call `mgr.Close()` explicitly in the exec path (just before `execFn()`) so
  all plugin subprocesses are terminated before the process image is replaced.
- The existing `defer mgr.Close()` in `main.go` remains as a safety net for
  error paths that return before reaching exec.

Non-goals:
- Re-using already-running plugins across `swm workspace open` invocations.
- Plugin lifetime tied to the tmux session or tmux server lifecycle.
- Any changes to how plugins are discovered, launched, or warmed.

## Capabilities

### New Capabilities

_(none)_

### Modified Capabilities

- `plugin-lifecycle`: Add a requirement that the host explicitly closes all
  plugins before calling `exec` to replace the process, so no plugin
  subprocess outlives the `swm workspace open` invocation.

## Impact

- `cmd/swm/internal/cli/workspace/open.go` — `pluginManager` interface gains
  `Close() error`; exec path calls `mgr.Close()`.
- `cmd/swm/internal/cli/workspace/open_test.go` — test double updated to
  implement `Close()`.
- No proto changes. No API surface changes. No other packages affected.

Capability surface affected: **none** (host-internal change only).
