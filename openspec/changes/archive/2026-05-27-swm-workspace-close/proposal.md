# Proposal: swm workspace close

## Why

`swm story remove` is the only way to close an active workspace today, but it also
deletes the story and all its worktrees. Users who want to stop the multiplexer
session and reclaim resources without destroying their work have no supported path —
they must kill the tmux server manually.

## What Changes

- Add `swm workspace close [<name>]` CLI command under the existing `swm workspace`
  sub-command group.
- The `<name>` argument is optional; when omitted the story name is resolved from
  `$SWM_STORY`. Missing name with unset `$SWM_STORY` exits non-zero.
- The command calls `session.ListWorkspaces`, finds the workspace matching the story
  name, then calls `session.CloseWorkspace`. If no active workspace is found the
  command succeeds (idempotent, workspace already closed).
- Shell completion for `<name>` lists active story names.

## Capabilities

### New Capabilities

_None._ The session plugin proto already exposes `CloseWorkspace`; no new RPC or
plugin capability is needed.

### Modified Capabilities

- **workflow-commands** — adds the `swm workspace close` command requirement. The
  existing `swm workspace open` requirement is unchanged.

## Impact

- `cmd/swm/internal/cli/workspace/` — new `close.go` file; `close` sub-command
  registered on the workspace cobra group.
- `cmd/swm/internal/cli/story/remove.go` — `closeStoryWorkspace` helper extracted
  to a shared location so both `swm story remove` and `swm workspace close` can
  reuse it without duplication.
- No proto changes. No new plugin binary. No config changes.
- Capability surface: **session** (read: ListWorkspaces; write: CloseWorkspace).

## Non-goals

- Closing all open workspaces at once (`--all` flag) — not in scope.
- Hooks (`pre-workspace-close` / `post-workspace-close`) — can be added later.
- Removing the story or its worktrees — that remains the exclusive domain of
  `swm story remove`.
