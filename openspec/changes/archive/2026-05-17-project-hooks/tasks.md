## 1. Production Code (cmd/swm)

- [x] 1.1 In `cmd/swm/internal/cli/workspace/open.go`, inside the `!isAttached` block in `openWithPicker`, add a `hooks.Run` call for `pre-worktree-create` (with `ProjectHost`, `ProjectPath`, `WorktreePath`, `RepoPath` set) before the `vcs.CreateWorktree` call; return an error if any hook exits non-zero
- [x] 1.2 After `store.Update` in that same block, add a `hooks.Run` call for `post-worktree-create` with the same project context; log failures with `slog.WarnContext` and continue

## 2. Tests (cmd/swm)

- [x] 2.1 In `cmd/swm/internal/cli/workspace/open_test.go`, add a table-driven test case for "pre-worktree-create hook aborts worktree creation": stub `hooks.Run` to return an error for `pre-worktree-create`, assert `vcs.CreateWorktree` is never called and the command exits non-zero
- [x] 2.2 Add a test case for "post-worktree-create hook fails — logged, open continues": stub `hooks.Run` to return an error for `post-worktree-create`, assert the workspace is still opened and the command exits 0
- [x] 2.3 Add a test case for "post-worktree-create hook receives correct context": assert that the `RunConfig` passed to `hooks.Run` for `post-worktree-create` has the correct `ProjectHost`, `ProjectPath`, `WorktreePath`, and `RepoPath` values

## 3. Verification

- [x] 3.1 Run `task fmt` and confirm zero diff
- [x] 3.2 Run `task lint` and confirm zero findings
- [x] 3.3 Run `task test` and confirm all tests pass
