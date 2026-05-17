# Design: swm story list

## Context

`swm story create` and `swm story remove` are implemented. `Store.List(ctx)`
already exists in the `JSONStore` and returns all stories sorted lexically. The
CLI has no command to surface that data. Users must inspect
`$XDG_DATA_HOME/swm/stories/` directly.

## Goals / Non-Goals

**Goals**
- Expose `Store.List` as `swm story list`.
- Print story names to stdout, one per line, in lexical order.

**Non-Goals**
- Formatting flags (`--json`, `--table`).
- Metadata columns (branch, created-at, attached projects).
- Filtering or sorting flags.
- Hooks — listing is a pure read; no side-effects warrant hook points.

## Decisions

### One name per line, no header

**Decision**: Print bare story names, one per line.

**Alternatives considered**:
- Tabular output (name + branch + date) — deferred to non-goals; adds output
  coupling before users have expressed a need.
- JSON output — same deferral rationale.

**Rationale**: Matches the shell-friendly output style of `swm pr list` and
keeps the implementation trivial. Future flags can add formatting without
breaking existing consumers.

### No hooks

**Decision**: No `pre-story-list` / `post-story-list` hook events.

**Rationale**: Hook events exist for operations with side-effects (file
creation, worktree manipulation, workspace transitions). A read-only list
command has no meaningful pre/post contract to offer hook scripts.

### Store wires in via existing constructor pattern

**Decision**: `NewListCmd(store coreStory.Store) *cobra.Command` — same
signature pattern as `NewCreateCmd`.

**Rationale**: Keeps dependency injection consistent across story sub-commands.
No new interfaces needed; `Store.List` is already part of `coreStory.Store`.

## Risks / Trade-offs

- `Store.List` always bootstraps `_default` on first call — users will always
  see at least one story. This is existing behaviour and acceptable.

## Migration Plan

No data-model changes. The new binary is a strict superset of the old one.
Rollback is replacing the binary.

## Open Questions

_(none)_
