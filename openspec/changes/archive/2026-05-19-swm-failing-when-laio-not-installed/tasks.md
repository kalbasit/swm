## 1. Test (Red)

- [x] 1.1 In `plugins/session-tmux/internal/session/tmux_test.go`, add a table-driven test case for `OpenPaneGroup` where `pane_group_command` names a binary not present in PATH — assert the call returns a `FailedPrecondition` gRPC error and no tmux session is created

## 2. Implementation (Green)

- [x] 2.1 In `plugins/session-tmux/internal/session/tmux.go`, add a helper `validateCommandBinary(cmd string) error` that extracts the first whitespace-separated token from `cmd` and calls `exec.LookPath` on it, returning a `codes.FailedPrecondition` gRPC status error naming the missing binary if not found
- [x] 2.2 In `OpenPaneGroup` (`plugins/session-tmux/internal/session/tmux.go`), after `initialCmd := t.paneGroupCommand(ctx, req)`, call `validateCommandBinary(initialCmd)` when `initialCmd` is non-empty and return the error immediately if it fails (before any `tmux has-session` call)

## 3. Verify

- [x] 3.1 Run `task fmt && task lint && task test` in `plugins/session-tmux/` — all must pass
- [x] 3.2 Run `task fmt && task lint && task test` at repo root — all must pass
