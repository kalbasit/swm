---
id: proposal
change: fix-default-story
---

## Why

When a user selects `_default` in the interactive story picker for a project not yet attached to that story, `swm workspace open` calls `vcs.CreateWorktree` with the canonical repo path as the worktree path. Because the canonical `repositories/` path is already the main git checkout, `git worktree add` fails with `fatal: '...' already exists`. The previous fix (a272404) correctly made `WorktreePath` return the canonical path for `_default`, but failed to skip the `CreateWorktree` call when the selected story is the default story.

## What Changes

- Skip `vcs.CreateWorktree` in `openWithPicker` when `storyName == cfg.DefaultStory`, since the canonical path is the main worktree and cannot be created again. Surrounding hooks and store attachment still execute.
- Still attach the project to the story store so subsequent opens skip this branch.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `workflow-commands` — adds a missing scenario to the `swm workspace open` requirement: selecting `_default` for an unattached project must NOT call `vcs.CreateWorktree`.

## Impact

- `cmd/swm/internal/cli/workspace/open.go` — `openWithPicker` function, lines ~344–401.
- `openspec/specs/workflow-commands/spec.md` — new scenario under `swm workspace open`.
- No proto changes. No new dependencies.
