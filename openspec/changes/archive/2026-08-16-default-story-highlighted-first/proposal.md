## Why

When `swm workspace open` is run with no argument, the story picker currently pins
`_default` as the **last** entry. Because the bundled fzf picker uses its default
layout (no `--layout=reverse`), the cursor starts on the **first** item sent — so a
recent feature story sits under the cursor and `_default` is furthest away. Opening
the default story (the common case) requires scrolling past every feature story.
`_default` should be the item under the cursor so a bare `swm workspace open` +
Enter opens it.

## What Changes

- Reorder the story picker so `_default` is the **first** entry (under the fzf
  cursor) instead of the last. Feature stories follow, still sorted by `CreatedAt`
  descending with a lexicographic tie-break.
- Update `SortStoriesForPicker` in `cmd/swm/internal/cli/workspace/sort_stories.go`
  to pin `_default` first.
- Update the `workflow-commands` spec: the "sorted … with `_default` pinned last"
  requirement and the "Story picker entries include _default as last entry"
  scenario become *first*.

Not a breaking change to any plugin surface — only the host's presentation order of
picker items changes.

## Capabilities

### New Capabilities

- (none)

### Modified Capabilities

- `workflow-commands`: the story-picker ordering requirement for
  `swm workspace open` changes so `_default` is presented first (under the cursor)
  rather than pinned last.

## Impact

- **Capability surface(s):** none. This is host-side CLI presentation ordering
  (`cmd/swm`); no plugin capability (session/vcs/forge/picker/hook) contract
  changes.
- **Proto changes:** none. No message shapes change, so no version bump under
  `proto/swm/plugin/vN/`.
- **Code:** `cmd/swm/internal/cli/workspace/sort_stories.go` and its test
  `sort_stories_test.go`; picker flow in `open.go` is unaffected (it consumes the
  sorted slice as-is).
- **Docs/TDD:** no TDD section governs story-picker ordering (see
  `docs/tdd/00001-plugin-host-architecture.md` for the host/plugin split); the
  behavior is defined solely by the `workflow-commands` spec, updated here.
- **Dependencies:** none.

## Non-goals

- No change to the **project** picker ordering (attached-projects-first) used after
  a story is selected.
- No change to fzf invocation flags (no `--layout=reverse`); cursor-on-first-item
  behavior is relied upon, not modified.
- No change to how `_default` is displayed (`_default (main repo)`).
- No change to the non-picker fallback resolution order (arg → `$SWM_STORY` →
  `_default`).
