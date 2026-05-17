## Why

`swm workspace open` switches the user into a target workspace but leaves the originating pane alive in the source session, creating dangling terminal state. v1 solved this with `swm tmux switch-client --kill-pane`; v2 needs the same behaviour as a first-class, multiplexer-agnostic capability so it works unchanged when zellij (or any future multiplexer) is added.

## What Changes

- Add `close_origin bool` field to `SwitchToRequest` proto message (proto change under `proto/swm/plugin/v1/` — backward-compatible proto3 field addition).
- Add `--kill-pane` flag to `swm workspace open`; when set, passes `close_origin: true` to `Session.SwitchTo`.
- Implement `close_origin` in `plugins/session-tmux`: capture the originating tmux pane reference before issuing `switch-client`, then `tmux kill-pane` on it after the switch completes.

## Non-goals

- Zellij integration (this change makes the proto and CLI ready for it; the actual zellij plugin comes later).
- Killing the entire source workspace/session — only the originating pane is closed.
- Any changes to how projects or worktrees are opened.

## Capabilities

### New Capabilities

_(none)_

### Modified Capabilities

- `session-tmux` — `SwitchTo` gains close-origin pane behaviour when `close_origin: true`.
- `workflow-commands` — `swm workspace open` gains `--kill-pane` flag wired to `SwitchToRequest.close_origin`.

## Impact

- `proto/swm/plugin/v1/session.proto` — new `close_origin bool` field on `SwitchToRequest`; regenerated Go code under `proto/`
- `plugins/session-tmux/` — `SwitchTo` RPC handler updated
- `cmd/swm/internal/cli/workspace/open.go` — `--kill-pane` flag added
