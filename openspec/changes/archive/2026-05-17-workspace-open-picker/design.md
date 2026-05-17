## Context

`swm workspace open` currently resolves a story by: positional arg → `$SWM_STORY` env var →
config `default_story` (`_default`). Users with multiple active stories must memorise or type
the exact story name every time. The host already has a picker capability for project selection
inside a story; this design extends that two-phase pattern with an up-front story-selection
phase.

Relevant code: `cmd/swm/internal/cli/workspace/open.go` — `openWithPicker` (story→project flow),
`openAllAttached` (fallback).

## Goals / Non-Goals

**Goals:**
- Show a story picker when `workspace open` is called with no story resolved from arg or env
- Display per-entry: story name, branch name (if different from story name), age (rounded up),
  attached projects (space-permitting, truncated to terminal width)
- `_default` story listed as `_default (main repo)` in display; raw `_default` used as key
- Stories sorted by `CreatedAt` descending (most recent first) in the picker
- Terminal-width-aware display strings — truncated in the host before streaming to picker

**Non-Goals:**
- Changes to the picker gRPC protocol or proto messages
- Changes to the project-within-story picker that runs after story selection
- Multi-select (single story selection only)
- Behaviour change when a positional arg or `$SWM_STORY` is present

## Decisions

### D1 — Terminal width detected in the host, not the picker plugin

**Decision:** The host opens `/dev/tty` to call `term.GetSize`, falls back to `$COLUMNS` env var
(parsed as int), then defaults to 120 columns. It truncates the display string before sending
each `PickItem` to the picker plugin.

**Rationale:** The picker plugin already receives opaque `display` strings; keeping truncation in
the host avoids leaking layout concerns into the plugin contract. `/dev/tty` is usable even when
stdout/stderr are redirected to fzf's pipe. `$COLUMNS` catches shells that set it but don't have
a PTY (e.g., multiplexer panes on some systems).

**Alternative considered:** Pass terminal width via gRPC metadata and let the plugin truncate.
Rejected — adds protocol coupling for a purely presentational concern.

### D2 — Story picker triggers only when no story is resolved from arg or env

**Decision:** The story picker runs only when the positional arg is absent AND `$SWM_STORY` is
unset (or empty). If either is present, the current flow is unchanged.

**Rationale:** `$SWM_STORY` is typically set by shell hooks that already encode the user's story
context; overriding it with a picker would be surprising. Users in a scripted context (env var
set) get deterministic behaviour; users at an interactive prompt get the picker.

### D3 — Two new internal packages; no package merging

**Decision:**
- `cmd/swm/internal/ageformat` — `FormatAge(t time.Time, now time.Time) string` returning
  rounded-up strings: `Xm ago`, `Xh ago`, `Xd ago`, `Xw ago`, `Xmo ago`, `Xy ago`.
- `cmd/swm/internal/termwidth` — `Detect() int` using `/dev/tty` → `$COLUMNS` → 120 fallback.

**Rationale:** One concept per package matches the project convention. Both utilities are small
and testable in isolation (age formatting uses a `now` parameter so tests don't depend on wall
time; terminal detection is trivially mockable by setting `$COLUMNS`).

### D4 — Truncation priority

When the formatted display string exceeds terminal width, the host trims right-to-left:

1. Projects list — trailing entries dropped, replaced with ` …` suffix
2. Branch name — truncated inside parens with `…`
3. Story name — last resort, truncated with `…`

A minimum guaranteed width is reserved for story name + age so the entry is never entirely
uninformative.

### D5 — Stories sorted by CreatedAt descending

Most recently created stories appear first. `_default` is always pinned last (it is the oldest
entry and the least likely intended target after the first setup).

## Risks / Trade-offs

| Risk | Mitigation |
|---|---|
| `/dev/tty` unavailable in rare environments (containers, some CI) | Fallback chain handles it; no TTY → `FailedPrecondition` from picker → `openAllAttached` fallback unchanged |
| Many stories make display hard to read | fzf fuzzy search mitigates this; no pagination needed |
| `term.GetSize` returns 0 columns on some systems | Clamp detected width: if ≤ 0 fall through to next fallback |

## Open Questions

None — all decisions above are settled based on the proposal and conversation context.
