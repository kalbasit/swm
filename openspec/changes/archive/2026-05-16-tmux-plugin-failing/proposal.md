## Why

After the XDG config-path fix, `swm workspace open` still fails — both session-tmux and
picker-fzf plugins start, report their gRPC addresses, then crash within milliseconds,
leaving only "connection reset by peer" / "EOF" in host logs. Because go-plugin silently
discards plugin stderr by default, there is no actionable diagnostic today.

## What Changes

- Route plugin stderr to host log output (DEBUG level) so plugin panics and runtime errors
  are visible without requiring a separate debug run.
- Identify and fix the root cause of the crash (hypothesis: plugins inherit only
  `SWM_HOST_SOCKET` in their environment, stripping `PATH`, `HOME`, and `XDG_*` vars
  needed by the session plugin at RPC-call time).

## Capabilities

### New Capabilities

_(none)_

### Modified Capabilities

- `workflow-commands`: the `swm workspace open` failure scenario now has an observable
  error path; the requirement that plugin crashes surface a human-readable message may
  need to be made explicit.

## Impact

- `cmd/swm/internal/pluginmgr/manager.go` — plugin `Cmd.Env` construction and
  `plugin.ClientConfig` (Stderr field).
- `plugins/session-tmux/` — may need a bug fix once stderr is readable.
- No proto changes required.
- No new capability surfaces; no breaking changes.
