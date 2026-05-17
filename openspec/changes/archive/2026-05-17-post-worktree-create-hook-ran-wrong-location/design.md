## Context

`hookexec.RunConfig` already carries `WorktreePath`, `RepoPath`, and `CodeRoot` for use as environment variables, but the executor builds the hook command with `exec.CommandContext` and never sets `Cmd.Dir`. As a result, every hook inherits the working directory of the swm process — typically wherever the user invoked the CLI — rather than a path meaningful to the hook script.

The fix is a single-field addition to `RunConfig` (`WorkDir string`) and a one-liner in the executor. All existing call sites are updated in the same change.

## Goals / Non-Goals

**Goals:**
- Hooks execute with a working directory appropriate to their event.
- Hook authors can rely on `$PWD` matching the documented run location without reading env vars.
- The hook system is documented in both root and host-CLI READMEs.

**Non-Goals:**
- No changes to hook discovery, tier resolution, environment variables, or stdin JSON.
- No new hook events.
- No validation that `WorkDir` exists before invoking hooks (OS error is sufficient).
- No changes to pre-/post-* failure semantics.

## Decisions

### Add `WorkDir string` to `RunConfig`; callers set it explicitly

**Alternatives considered:**
- *Infer from event name inside the executor*: hookexec would need a mapping from event → which path field to use. This couples the executor to event semantics and makes it harder to add new events or override the directory in tests.
- *Derive from existing path fields + event name at call site*: same coupling, just moved to the caller.

**Decision:** A plain `WorkDir string` field on `RunConfig`. Each call site sets it to the appropriate path from the values it already has. The executor does one thing: `cmd.Dir = cfg.WorkDir` (when non-empty). Policy lives with the caller; the executor stays dumb.

### Fall back to inherited cwd when `WorkDir` is empty

All known call sites will be updated, but leaving the fallback as "inherit cwd" (rather than returning an error on empty `WorkDir`) keeps the change non-breaking for any future call sites added before they set `WorkDir`, and avoids a hard failure during the transition.

### `WorkDir` per event

| Event | WorkDir | Rationale |
|---|---|---|
| `pre-story-create` | `codeRoot` | no repo context |
| `post-story-create` | `codeRoot` | no repo context |
| `pre-story-remove` | `codeRoot` | no repo context |
| `post-story-remove` | `codeRoot` | no repo context |
| `pre-worktree-create` | `repoPath` | worktree not yet created; repo exists |
| `post-worktree-create` | `worktreePath` | worktree was just created |
| `pre-worktree-remove` | `worktreePath` | last chance to act inside the worktree |
| `post-worktree-remove` | `repoPath` | worktree gone; repo still present (e.g. `git worktree prune`) |
| `pre-clone` | `codeRoot` | repo does not exist yet |
| `post-clone` | `repoPath` | newly cloned repo |
| `pre-workspace-open` | `worktreePath` | opening into the worktree |
| `post-workspace-open` | `worktreePath` | opened into the worktree |

## Risks / Trade-offs

- **`pre-worktree-create` with a non-existent repo** → If the repo has never been cloned, `repoPath` won't exist. The OS will fail to start the hook process. This is intentional: hook authors should only place a `pre-worktree-create` hook in a per-repo tier if the repo exists. Global and per-story hooks for this event would need to check `$SWM_REPO_PATH` themselves. Acceptable trade-off; the alternative (silently falling back to cwd) is worse than an obvious failure.

- **Existing hook scripts that relied on the inherited cwd** → Breaking change in behavior, but those scripts were relying on undefined/accidental behavior. The new behavior is correct and documented.

## Migration Plan

All changes ship in a single commit set:
1. Add `WorkDir` to `RunConfig` and set `cmd.Dir` in the executor.
2. Update all call sites (`clone.go`, `story/create.go`, `story/remove.go`, `workspace/open.go`).
3. Add spec scenarios and update `cmd/swm/README.md` and root `README.md`.

No rollback concern — the change is confined to the swm host binary with no protocol or plugin API impact.
