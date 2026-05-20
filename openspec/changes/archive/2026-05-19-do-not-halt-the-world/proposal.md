## Why

`Warm()` is called in `PreRunE` and blocks until every requested plugin has started,
even though plugins start concurrently with each other. The user-visible latency is
identical to lazy startup — we just pay it earlier and with no feedback.
The fix is to make warming genuinely non-blocking: fire startup in the background
and let the first actual `Get()` call be the natural synchronization point.

## What Changes

- `Manager.Warm()` becomes fire-and-forget: it launches a startup goroutine per
  capability and returns immediately (no `wg.Wait()`).
- `Manager.Get()` becomes the wait point: if the plugin for a capability is still
  booting, `Get()` blocks until it is ready (or errors).
- `PreRunE` hooks are retained — they continue to call `Warm()` to initiate early
  startup, but they no longer stall command execution.
- The first-error-cancels-rest semantics move from `Warm()` into `Get()`: a failed
  startup is surfaced to the first caller that needs the plugin, not to PreRunE.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None — this is an internal change to `pluginmgr.Manager`. No plugin-facing
capability surface (`session`, `vcs`, `forge`, `picker`, `hook`) changes its
observable behavior or protocol.

## Non-goals

- Changing the plugin gRPC protocol or any proto definitions.
- Removing `Warm()` call sites — they remain as hints to start early.
- Surfacing progress/spinner UI during plugin startup.
- Changing how plugin errors are ultimately reported to the user.

## Impact

- `cmd/swm/internal/pluginmgr/Manager`: `Warm()` and `Get()` implementations change.
- All `PreRunE` hooks that call `Warm()`: behavior changes (no longer blocks), but
  call sites are unchanged.
- No proto changes; no capability surface changes.
- Capability surface(s) affected: **none**.
