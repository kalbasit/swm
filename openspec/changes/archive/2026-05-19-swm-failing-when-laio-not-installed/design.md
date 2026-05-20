## Context

The `session-tmux` plugin's `OpenPaneGroup` accepts an optional `pane_group_command` from the host config. This command (e.g. `laio start ...`) is passed as the initial command to `tmux new-session`. When the binary in that command is missing from PATH, tmux creates the session but the command exits immediately — the user sees `[exited]` with no actionable error.

The fix is localized to `plugins/session-tmux/internal/session/tmux.go`. No proto, config schema, or inter-plugin changes are needed.

## Goals / Non-Goals

**Goals**
- Surface a clear error before the tmux session is created when `pane_group_command` names a binary not in PATH.

**Non-Goals**
- Validating arguments beyond the binary name.
- Detecting runtime failures (the command launches but crashes later).
- Changing how PATH is resolved (standard `exec.LookPath` only).

## Decisions

### Parse first token and call `exec.LookPath`

In `OpenPaneGroup`, after `paneGroupCommand` returns a non-empty command string, split on whitespace to extract the first token and call `exec.LookPath` on it. If not found, return `codes.FailedPrecondition` with a message that names the missing binary and suggests the user check their PATH or install the tool.

**Alternative considered**: shell-validate via `command -v` inside a subprocess — rejected because it adds a shell dependency and is slower than an in-process lookup.

**Alternative considered**: let tmux report the failure and parse its stderr — rejected because tmux does not return a non-zero exit for "command not found in new-session"; it creates the session and lets the window exit.

### Where to add the check

In `OpenPaneGroup` immediately after `initialCmd := t.paneGroupCommand(ctx, req)` and before the `tmux has-session` call. A helper `validateCommandBinary(cmd string) error` keeps the logic separate and testable.

### gRPC error code

`codes.FailedPrecondition` — the request is valid; the environment is not ready. This maps to HTTP 400 and is surfaced as a user-visible error by the host CLI.

## Risks / Trade-offs

- **PATH divergence**: the plugin runs in the same process environment as `swm`, so if `laio` is in the user's PATH, `LookPath` finds it. If the user's shell profile adds a custom path not visible to `swm`, validation might reject a binary that would work inside the tmux session. This is the same limitation tmux itself has. → Acceptable; the common case (binary not installed at all) is caught cleanly.

## Migration Plan

No migration needed. The change is purely additive: new error path that was previously a silent failure.

## Open Questions

_(none)_
