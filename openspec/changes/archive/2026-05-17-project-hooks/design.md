## Context

The `hookexec` package is fully implemented and all commands that manage stories and clones already call it correctly. The `workflow-commands` spec also documents the required hook calls. However, the worktree creation path inside `openWithPicker` (`cmd/swm/internal/cli/workspace/open.go`, the `!isAttached` block around line 262) calls `vcs.CreateWorktree` with no surrounding hook invocations. This is the only missing wiring.

The debug log from `swm --log-level debug workspace open story-a` confirms it: the session plugin opens, the picker runs, a pane group is opened — but no hookexec messages appear, even though a new worktree was created (story had 0 projects before the run).

## Goals / Non-Goals

**Goals:**
- Add `pre-worktree-create` hook call before `vcs.CreateWorktree` in `openWithPicker`; abort if any hook exits non-zero.
- Add `post-worktree-create` hook call after `store.Update` in `openWithPicker`; log failures and continue.

**Non-Goals:**
- Changing the hookexec package itself (discovery logic, env vars, event list, stdin JSON).
- Adding worktree-create hooks to `openAllAttached` — that path opens already-attached projects and never calls `vcs.CreateWorktree`.
- Adding debug log statements around hook calls (the hookexec package already logs at warn on failure; individual commands don't need extra instrumentation).
- Wiring any other commands — story create/remove, clone, and workspace open/close already call hookexec correctly.

## Decisions

### 1. Only `openWithPicker` needs changes

`openAllAttached` iterates projects that are already attached (worktrees already exist). It never calls `vcs.CreateWorktree`, so no worktree-create hooks belong there.

`story.NewRemoveCmd` already calls `pre-worktree-remove` / `post-worktree-remove` in its removal loop.

### 2. Hook call placement

```
pre-worktree-create  →  vcs.CreateWorktree  →  store.Update  →  post-worktree-create
```

- `pre-worktree-create` runs before the VCS call so that a hook can abort before disk state changes.
- `post-worktree-create` runs after `store.Update` so that the hook observes the fully-committed state (worktree on disk + story JSON updated).

### 3. Context fields to pass

All fields are available at the call site:
- `ProjectHost`: `pid.GetHost()`
- `ProjectPath`: `strings.Join(pid.GetSegments(), "/")`
- `WorktreePath`: `worktreePath` (already computed above the block)
- `RepoPath`: `resolver.CanonicalPath(pid)` (same value passed to CreateWorktree)

### 4. Pre-hook failure aborts the command

Consistent with every other `pre-*` hook site in the codebase: if `pre-worktree-create` returns an error, `openWithPicker` returns that error immediately, without calling `vcs.CreateWorktree`.

### 5. Post-hook failure is logged, not fatal

Consistent with every other `post-*` hook site: `hookexec` already logs the failure; `openWithPicker` continues to `OpenWorkspace`.

## Risks / Trade-offs

- **No risk of double-firing**: The `!isAttached` guard is the only code path that calls `vcs.CreateWorktree` inside workspace open. The change is surgical.
- **Hook script not executable**: `hookexec.findHooks` already skips non-executable files, so a script without `chmod +x` silently does nothing (matches existing behavior across all hook tiers).
- **Pre-hook abort leaves no partial state**: The worktree has not been created yet when `pre-worktree-create` runs, so aborting is safe.
