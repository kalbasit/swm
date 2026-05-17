# Design: session-name

## Context

The `session-tmux` plugin derives tmux session names from a `ProjectID` in two
places:

1. **`OpenPaneGroup`** (line 191, `tmux.go`): `name := segments[len(segments)-1]`
   — hardcodes the last path segment.
2. **`sessionName(key string)`** (line 347, `tmux.go`): splits on `/` and
   returns the last part — same bug, called from `OpenWorkspace`.

Both paths produce the same collision: `github.com/a/repo` and
`github.com/b/repo` both yield session name `repo`.

The `ProjectID` proto already carries `host` and `segments`, and the
`worktree_paths` map keys in `OpenWorkspaceRequest` are already in
`host/seg1/.../segN` form. No data is missing — only the derivation is wrong.

## Goals / Non-Goals

**Goals:**
- Session names uniquely identify a project across all forges and orgs.
- `sessionName()` is the single source of truth for the derivation; both
  `OpenWorkspace` and `OpenPaneGroup` go through it.
- The sanitization lives entirely in the plugin; the host and proto are
  unchanged.

**Non-Goals:**
- Hashing as an alternative (deferred; tmux name length is not yet a
  practical concern).
- User-configurable name templates or aliases.
- Migrating existing workspaces automatically (users close and reopen).

## Decisions

### 1. Use the full `host/seg1/.../segN` key as the session name basis

**Decision:** `sessionName(key string)` returns the sanitized full key, not its
last segment.

**Rationale:** The key already uniquely identifies a project. Slashes are valid
in tmux session names. Only dots and colons need replacement (dots collide with
tmux's `:` separator in some contexts; colons are reserved in the tmux target
syntax).

**Alternative considered:** Use only `host + "/" + last-segment` to keep names
shorter. Rejected because it still collides when two orgs on the same host have
repos with the same name (e.g. `github.com/a/utils` vs `github.com/b/utils`).

### 2. Sanitize `.` → `•` (U+2022) and `:` → `：` (U+FF1A)

**Decision:** Replace ASCII dot with bullet and ASCII colon with fullwidth
colon, matching the strategy used in swm-v1.

**Rationale:** These are visually distinguishable, round-trip safely through
tmux's session name handling, and are already familiar to existing users.
Slashes are left as-is because tmux accepts them and they aid readability.

**Alternative considered:** Percent-encoding or base64. Rejected because the
result is unreadable in `tmux ls` output.

### 3. Fix `OpenPaneGroup` to construct the full key before calling `sessionName`

**Decision:** Replace `name := segments[len(segments)-1]` with
`key := pid.GetHost() + "/" + strings.Join(pid.GetSegments(), "/")` followed by
`name := sessionName(key)`.

**Rationale:** `OpenWorkspace` already receives keys in `host/seg/.../last`
form (from the `worktree_paths` map). Making `OpenPaneGroup` construct the same
key lets both paths share the same `sessionName` function without divergence.

### 4. `pane_group_id` in the returned `PaneGroup` proto changes

**Decision:** `PaneGroup.pane_group_id` will be the sanitized full path (e.g.
`github•com/kalbasit/swm`) rather than `swm`.

**Rationale:** `pane_group_id` is used as the tmux target in `SwitchTo`; it
must match the actual session name. Callers that store or display this value
will see a different string after the upgrade.

## Risks / Trade-offs

- **Stale session names** → Existing workspaces opened before the upgrade will
  have sessions named after the old (basename-only) scheme. `SwitchTo` will
  fail to find them. Mitigation: document that users must close and reopen all
  workspaces after upgrading.
- **`pane_group_id` is an opaque identifier** → Callers should treat it as
  opaque and not parse it. This is already the contract, so the change is
  backward-compatible at the interface level, but not at the data level.

## Migration Plan

1. Upgrade `swm-plugin-session-tmux` binary.
2. Run `swm workspace close` on all open workspaces (or kill tmux servers
   manually: `tmux -S <socket> kill-server`).
3. Run `swm workspace open` to reopen — new session names will be applied.

No config changes required. No proto changes required.

## Open Questions

_None._
