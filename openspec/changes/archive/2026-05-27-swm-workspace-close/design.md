# Design: swm workspace close

## Context

`swm story remove` already contains a `closeStoryWorkspace` helper (unexported, in
`cmd/swm/internal/cli/story/remove.go`) that: streams `ListWorkspaces`, matches by
story name, then calls `CloseWorkspace`. That logic is exactly what the new command
needs, but it is currently buried in the story package with best-effort semantics
(errors are silently dropped).

The session plugin proto already has `CloseWorkspace(CloseWorkspaceRequest)` and the
`session-tmux` spec already requires idempotency (close on a missing workspace
succeeds). No proto changes or plugin changes are needed.

The workspace package (`cmd/swm/internal/cli/workspace/`) already owns `open.go` and
`list.go`; `close.go` is the natural home for this command.

## Goals / Non-Goals

**Goals**
- Add `swm workspace close [<name>]` that tears down the running multiplexer for a
  story without touching the story record or worktrees.
- Default to `$SWM_STORY` when no name is given; error when both are absent.
- Succeed (idempotent) when no workspace is active for that story.
- Shell-complete `<name>` from the story store.

**Non-Goals**
- Hooks (`pre-workspace-close` / `post-workspace-close`).
- A `--all` flag or closing multiple workspaces.
- Removing the story or its worktrees (that remains `swm story remove`).

## Decisions

### 1. Extract `closeStoryWorkspace` to the workspace package

**Decision:** Move the close logic into `cmd/swm/internal/cli/workspace/close.go` and
update `story/remove.go` to call it from there (or duplicate the small helper).

**Why:** The workspace package is the domain owner; placing close logic there is
consistent with how `open.go` and `list.go` are structured. Keeping it in the story
package would mean the workspace command reaches into a sibling package, inverting the
dependency direction.

**Alternative considered:** Re-export from `story` package. Rejected — circular
imports are not possible here but the logical ownership is wrong; story commands should
not be a library for workspace commands.

**Practical approach:** The helper is ~15 lines. Duplicate it in
`cmd/swm/internal/cli/workspace/close.go` and keep `story/remove.go`'s private copy
as-is. Both are internal packages; DRY across `internal/` siblings adds more coupling
than it removes. Revisit if the logic grows.

### 2. Error semantics: fatal vs. best-effort

**Decision:** Errors from `ListWorkspaces` and `CloseWorkspace` are returned to the
caller (fatal), unlike `story remove` which silently drops them.

**Why:** `swm workspace close` is an explicit user action. When the user asks to close
a workspace and it fails, they need to know. Best-effort silence is appropriate in
`story remove` because workspace closure is a cleanup step subordinate to the primary
goal of deleting the story.

**Exception:** No active workspace (story not found in `ListWorkspaces`) is treated as
success (idempotent), matching the session-tmux spec and `story remove` behavior.

### 3. Workspace-ID resolution via `ListWorkspaces` (no new RPC)

**Decision:** Resolve workspace_id by streaming `ListWorkspaces` and filtering by
`story_name`, identical to how `story remove` does it.

**Why:** The proto's `CloseWorkspaceRequest` requires a `workspace_id`, not a
`story_name`. There is no `FindWorkspaceByStory` RPC and adding one would be over-
engineering for this use case. `ListWorkspaces` is a stream and returns quickly (only
active sockets).

**Alternative considered:** Add a `CloseWorkspaceByStory` RPC to the proto. Rejected —
the proto is stable and adding an RPC for a convenience mapping belongs at the host
layer, not the plugin contract.

## Risks / Trade-offs

- [Race condition] A workspace could be closed between `ListWorkspaces` and
  `CloseWorkspace`. → `CloseWorkspace` on a non-existent socket is idempotent
  (session-tmux spec, scenario "Close non-existent workspace"), so this is safe.
- [Session plugin absent] If no session plugin is configured, the command returns an
  error. → Acceptable; the user can't close a workspace they can't open.

## Open Questions

_None. All design decisions are resolved._
