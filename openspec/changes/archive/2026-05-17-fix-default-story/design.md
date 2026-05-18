---
id: design
change: fix-default-story
---

## Context

`swm workspace open` resolves the worktree path via `resolver.WorktreePath(storyName, pid)`. Since a272404, `WorktreePath` correctly returns the canonical `repositories/<host>/<path>` for the default story instead of `stories/_default/<host>/<path>`. However, `openWithPicker` still calls `vcs.CreateWorktree` unconditionally for any project not yet attached to the selected story. For `_default` this issues `git worktree add -b _default <canonical-path>`, which fails because the canonical path is already the main git checkout.

## Goals / Non-Goals

**Goals:**
- `swm workspace open` succeeds when the user selects `_default` for a project not yet attached to it.
- Pre/post `worktree-create` hooks still fire (they perform useful side-effects unrelated to git).
- The project is still attached to the story store so future opens are no-ops.

**Non-Goals:**
- Changing hook semantics for the default story.
- Changing `WorktreePath` or `CanonicalPath` logic (already correct).
- Handling the no-picker fallback path (it only reads already-attached projects, so the bug doesn't apply there).

## Decisions

**Guard `vcs.CreateWorktree` with a story-name check.**

In `openWithPicker`, wrap the `vcs.CreateWorktree` call (and the VCS plugin load that precedes it) with:

```go
if storyName != cfg.DefaultStory {
    // load VCS plugin and call CreateWorktree
}
```

Hooks and store attachment remain outside the guard and always execute.

*Why not check by path equality?* The story name is already available and is the canonical signal. A path comparison would couple layout logic into the CLI layer unnecessarily.

*Why not move the guard into `vcs.CreateWorktree` itself?* The VCS plugin is unaware of the default-story concept; keeping that concern in the CLI caller is consistent with the existing architecture.

## Risks / Trade-offs

- **Hooks run without a worktree being created** — for `_default` the canonical path already exists, so hook scripts that inspect the worktree path will find it present. This is correct behavior.
- **Minimal scope** — only `openWithPicker` is touched; the no-picker path is unaffected.

## Migration Plan

No migration needed. The change is a runtime guard; no data, schema, or config changes.
