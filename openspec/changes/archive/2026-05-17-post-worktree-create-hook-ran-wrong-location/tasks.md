## 1. hookexec — Add WorkDir field (cmd/swm)

- [x] 1.1 Add `WorkDir string` field to `hookexec.RunConfig` in `cmd/swm/internal/hookexec/hookexec.go`
- [x] 1.2 Set `cmd.Dir = cfg.WorkDir` (when non-empty) in the hook process builder in `cmd/swm/internal/hookexec/hookexec.go`
- [x] 1.3 Add table-driven test scenarios in `cmd/swm/internal/hookexec/hookexec_test.go` asserting hook runs in the correct working directory when `WorkDir` is set
- [x] 1.4 Add test scenario asserting hook inherits process cwd when `WorkDir` is empty

## 2. Call sites — Populate WorkDir (cmd/swm)

- [x] 2.1 `cmd/swm/internal/cli/story/create.go`: set `WorkDir = codeRoot` on both `pre-story-create` and `post-story-create` `RunConfig` values
- [x] 2.2 `cmd/swm/internal/cli/story/remove.go`: set `WorkDir = codeRoot` on `pre-story-remove` and `post-story-remove`; set `WorkDir = worktreePath` on `pre-worktree-remove`; set `WorkDir = repoPath` on `post-worktree-remove`
- [x] 2.3 `cmd/swm/internal/cli/workspace/open.go`: set `WorkDir = repoPath` on `pre-worktree-create`; set `WorkDir = worktreePath` on `post-worktree-create`, `pre-workspace-open`, and `post-workspace-open`
- [x] 2.4 `cmd/swm/internal/cli/clone.go`: set `WorkDir = codeRoot` on `pre-clone`; set `WorkDir = repoPath` on `post-clone`

## 3. Tests — Verify working directory per event (cmd/swm)

- [x] 3.1 Add or extend integration/unit tests in `workspace/open_test.go` confirming `post-worktree-create` hooks receive the worktree as cwd
- [x] 3.2 Add or extend tests in `story/remove_test.go` confirming `pre-worktree-remove` receives worktree cwd and `post-worktree-remove` receives repo cwd

## 4. Documentation — cmd/swm README

- [x] 4.1 Add a `## Hook System` section to `cmd/swm/README.md` covering: all 12 supported events with their working directory, tier resolution order (global → per-repo → per-story), environment variables (`SWM_HOOK`, `SWM_STORY`, `SWM_PROJECT_HOST`, `SWM_PROJECT_PATH`, `SWM_WORKTREE_PATH`, `SWM_REPO_PATH`), and stdin JSON contract

## 5. Documentation — Root README

- [x] 5.1 Add a brief hook system mention to the root `README.md` with a link to `cmd/swm/README.md#hook-system`
