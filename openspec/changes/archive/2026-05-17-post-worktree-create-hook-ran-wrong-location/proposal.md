## Why

Hook executables currently inherit the working directory of the swm process rather than the contextually appropriate path, so hooks like `post-worktree-create` run in an arbitrary directory instead of inside the new worktree. Users have no documented reference for what hooks exist, when they fire, or where they run.

## What Changes

- Add a `WorkDir` field to `hookexec.RunConfig`; the hook executor SHALL set `cmd.Dir` to this value when non-empty.
- Each call site SHALL populate `WorkDir` according to the table below:

  | Event | WorkDir |
  |---|---|
  | `pre-story-create` | `codeRoot` |
  | `post-story-create` | `codeRoot` |
  | `pre-story-remove` | `codeRoot` |
  | `post-story-remove` | `codeRoot` |
  | `pre-worktree-create` | `repoPath` |
  | `post-worktree-create` | `worktreePath` |
  | `pre-worktree-remove` | `worktreePath` |
  | `post-worktree-remove` | `repoPath` |
  | `pre-clone` | `codeRoot` |
  | `post-clone` | `repoPath` |
  | `pre-workspace-open` | `worktreePath` |
  | `post-workspace-open` | `worktreePath` |

- Add a Hook System section to `cmd/swm/README.md` covering: all supported events, their working directory, tier resolution order, environment variables, and stdin JSON contract.

## Capabilities

### New Capabilities

_(none)_

### Modified Capabilities

- `hook-executor`: add `WorkDir` field to `RunConfig`; executor sets `cmd.Dir` when non-empty; spec gains a working-directory requirement and scenarios per event class.
- `project-documentation`: `cmd/swm/README.md` must include a Hook System section (event reference table + tier resolution + env vars + stdin JSON). The root `README.md` must include a brief hook system mention with a link to `cmd/swm/README.md#hook-system`.

## Impact

- `cmd/swm/internal/hookexec/hookexec.go` — set `cmd.Dir = cfg.WorkDir` when non-empty.
- `cmd/swm/internal/hookexec/hookexec_test.go` — add scenarios asserting correct working directory per event class.
- `cmd/swm/internal/cli/workspace/open.go` — populate `WorkDir` on all four `RunConfig` values.
- `cmd/swm/internal/cli/story/create.go` — populate `WorkDir = codeRoot` on both `RunConfig` values.
- `cmd/swm/internal/cli/story/remove.go` — populate `WorkDir` on all four `RunConfig` values.
- `cmd/swm/internal/cli/clone.go` — populate `WorkDir` on both `RunConfig` values.
- `cmd/swm/README.md` — new Hook System section.
- `README.md` — brief hook system mention with link to `cmd/swm/README.md#hook-system`.
- No proto changes. No plugin API changes.
