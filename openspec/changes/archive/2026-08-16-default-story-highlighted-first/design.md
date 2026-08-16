## Context

`swm workspace open` (no arg) resolves a story via an interactive story picker. The
host builds the picker list in `SortStoriesForPicker`
(`cmd/swm/internal/cli/workspace/sort_stories.go`) and streams each story to the
picker plugin in that order (`pickStory` in `open.go`).

The bundled fzf plugin invokes fzf with no `--layout=reverse`
(`plugins/picker-fzf/internal/picker/fzf.go:92`). In fzf's default layout the cursor
starts on the **first item received**. Today `SortStoriesForPicker` pins `_default`
**last**, so a recent feature story sits under the cursor; opening the default story
(the common case) requires scrolling to the far end of the list.

This is a small, host-local presentation change. No plugin protocol, proto message,
or fallback-resolution behavior changes.

## Goals / Non-Goals

**Goals:**
- Make `_default` the first item streamed to the picker so it is under the cursor for
  a bare `swm workspace open` + Enter.
- Preserve the existing ordering of feature stories (CreatedAt descending, name
  tie-break).

**Non-Goals:**
- Changing the project picker (post-story-selection) ordering.
- Changing fzf invocation flags.
- Changing `_default` display text or the non-picker fallback path.

## Decisions

**Decision: Invert the pin in `SortStoriesForPicker` (first instead of last).**
The comparator already special-cases `_default`; only the sign flips — `_default`
sorts *before* any non-default story, and the existing CreatedAt-descending /
name-ascending comparison governs the rest. Minimal, localized diff in the one
function whose sole job is picker ordering.

- *Alternative — sort in `pickStory`/`open.go`:* rejected. `SortStoriesForPicker` is
  the dedicated, unit-tested seam for picker ordering; putting ordering logic in the
  caller would split the concern and duplicate the `_default` special-case.
- *Alternative — pass `--layout=reverse` / an fzf cursor flag to move the cursor to
  the list end instead of reordering:* rejected. That is a picker-plugin surface
  change affecting every picker use (project picker too), and couples host intent to
  one plugin's flags. Host-side ordering keeps the fix plugin-agnostic.

**Decision: Rely on "first item = under cursor" as the contract, not fzf internals.**
The requirement is expressed as "pinned first"; the fzf default-layout behavior is
the reason, documented in the spec. If a future picker places the cursor elsewhere,
that is a picker-spec concern, not this change's.

## Risks / Trade-offs

- **[A user with muscle memory expects `_default` last] → ** low impact; the whole
  point of the change is that the common target should be the default cursor
  position. No data or destructive behavior involved.
- **[A non-default picker plugin might not put the cursor on the first item] → **
  ordering is still correct and deterministic; only the "under cursor" ergonomic
  depends on picker layout, which the spec now states explicitly.

## Migration Plan

None required. Pure in-process ordering change; no state, config, or protocol
migration. Rollback is reverting the single-function change (and its test/spec).
