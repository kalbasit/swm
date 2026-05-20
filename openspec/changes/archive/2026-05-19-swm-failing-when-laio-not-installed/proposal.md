## Why

When `pane_group_command` references a binary not in PATH (e.g. `laio`), the session-tmux plugin creates a tmux session successfully but the command exits immediately, leaving the user with a silent `[exited]` and no actionable error message.

## What Changes

- In `OpenPaneGroup`, after resolving `pane_group_command`, extract the first token and verify it exists in PATH via `exec.LookPath`. Return a descriptive gRPC error if the binary is missing.
- No changes to proto, config schema, or other plugins.

## Capabilities

### New Capabilities

_(none)_

### Modified Capabilities

- `session-tmux` — add a new failure scenario to the existing `OpenPaneGroup` requirement: when `pane_group_command` is configured but its binary is not found in PATH, `OpenPaneGroup` SHALL return an error before creating the tmux session.

## Non-goals

- Validating the full pane_group_command argument list (only the binary is checked).
- Modifying how tmux handles commands that exit after the session starts (runtime failures after successful launch are out of scope).
- Any change to how the binary is resolved (PATH-based lookup only; no config override).

## Impact

- `plugins/session-tmux/internal/session/tmux.go` — `OpenPaneGroup` and `paneGroupCommand` (or a new helper).
- `openspec/specs/session-tmux/` — delta spec for the new error scenario.
- No API, proto, or config changes.
