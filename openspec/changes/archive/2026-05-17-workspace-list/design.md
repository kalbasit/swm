## Context

`swm workspace list` is a new read-only subcommand under the existing `workspace` cobra command group. It reads all stories from the story store, then renders them as a two-level tree (workspace → projects). There are no plugin invocations, no proto changes, and no mutations to any state.

## Goals / Non-Goals

**Goals**
- Print a pretty tree of workspaces and their attached projects.
- Be fast: a single store read, then in-memory sorting and rendering.

**Non-Goals**
- Live session state (whether a tmux socket is open).
- `--json` / machine-readable output.
- Filtering by workspace name or project.

## Decisions

### Tree rendering: hand-rolled, no library

The tree is exactly two levels deep and uses a fixed glyph set. A third-party tree-rendering library would add a dependency for a trivial format. A small `renderTree` helper in the command file is sufficient and keeps the code readable.

Alternatives considered:
- `github.com/xlab/treeprint` — unnecessary complexity for a 2-level tree.
- Recursive renderer — not needed; the structure is always `workspace → []project`.

### Glyphs and formatting

```
<workspace-name>
├── <project-path>
└── <project-path>
```

Box-drawing glyphs (`├──`/`└──`) clearly indicate tree membership. Workspaces with no projects omit the child lines entirely.

### Sorting

Stories are sorted lexicographically by name using `sort.Strings`. Projects within each story are sorted lexicographically by canonical path (`host + "/" + strings.Join(segments, "/")`). This matches user expectation and is deterministic for tests.

### Story store access

Use the same `StoryStore` interface already injected into other commands (e.g. `story list`). No new interfaces or abstractions needed.

### Color

Plain text only for the initial implementation. Terminal color is a cosmetic enhancement deferred to a future change.

## Risks / Trade-offs

- Large story counts render synchronously — acceptable because story counts are expected to be O(tens), not O(thousands).
- No pagination — intentional; this is a quick-glance tool.

## Migration Plan

No migration required. New command only; no existing behavior changes.

## Open Questions

None.
