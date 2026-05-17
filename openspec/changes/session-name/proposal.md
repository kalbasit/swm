# Proposal: session-name

## Why

The `session-tmux` plugin derives pane-group session names from the repository
basename (last path segment), causing silent collisions when two repositories
from different forges or orgs share the same name (e.g. `github.com/a/repo` vs
`github.com/b/repo` both map to session `repo`). The full canonical path must
be used instead.

## What Changes

- `sessionName()` in `plugins/session-tmux` is changed to sanitize and return
  the full `host/seg1/.../segN` key rather than just the last segment.
- `OpenPaneGroup` is changed to construct the full key
  (`host + "/" + strings.Join(segments, "/")`) before deriving the session name,
  consistent with how `OpenWorkspace` builds its `worktree_paths` keys.
- The `session-tmux` spec scenario for "Pane group session name" is updated to
  assert the full sanitized path is used.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `session-tmux` — the rule for how a pane-group session name is derived from
  a `ProjectID` changes at the spec level (new behavior, not just implementation
  detail).

## Non-goals

- Changing the on-disk worktree layout (host-owned invariant).
- Adding user-configurable name templates or aliases.
- Implementing hashing as an alternative sanitization strategy (left to a future
  change if tmux name length limits become an issue).
- Changing how the workspace socket is named (story-based, unaffected).
- Supporting non-tmux session plugins — this is a `session-tmux`-only change.

## Impact

- `plugins/session-tmux/internal/session/tmux.go` — `sessionName()` and
  `OpenPaneGroup`.
- `openspec/specs/session-tmux/spec.md` — "Pane group session name" scenario
  updated.
- No proto changes. `ProjectID.host` + `ProjectID.segments` already carry all
  necessary data; `worktree_paths` map keys already use the full path.
- Existing tmux workspaces opened before this change will have stale session
  names; users must close and reopen workspaces after upgrading.
