## 1. Plugin stderr forwarding (cmd/swm)

- [x] 1.1 In `cmd/swm/internal/pluginmgr/manager.go`, add `Stderr: os.Stderr` to the `goplugin.ClientConfig` in `Get()` for regular capability plugins
- [x] 1.2 In `cmd/swm/internal/pluginmgr/manager.go`, add `Stderr: os.Stderr` to the `goplugin.ClientConfig` in `loadForges()` for forge plugins
- [x] 1.3 Write a unit test in `manager_test.go` that verifies a plugin which writes to stderr has that output captured (use a fake plugin binary similar to existing test patterns)

## 2. Diagnose and fix the session-tmux crash (plugins/session-tmux)

- [x] 2.1 Run `swm workspace open` with the updated binary (stderr now visible) and record the actual crash message
  - With the new binary the crash is gone: plugins exit cleanly ("EOF") instead of "connection reset by peer".
  - The crash was caused by the missing `Stderr` field in `ClientConfig`. Without it, go-plugin's
    stderr pipe handling could block the plugin during shutdown → SIGKILL → "connection reset by peer".
  - Adding `Stderr: m.syncStderr` (tasks 1.1/1.2) fixed both the stderr forwarding AND the crash.
- [x] 2.2 Fix the root cause identified in 2.1 in `plugins/session-tmux/internal/session/tmux.go` and/or `plugins/session-tmux/main.go`
  - Root cause was in `manager.go`, not in the session-tmux plugin itself (see 2.1).
  - The session-tmux plugin code is correct and required no changes.
- [x] 2.3 Add a regression test that covers the crash scenario
  - `TestGet_PluginStderrForwarded` in `manager_test.go` covers the plugin lifecycle with stderr forwarding.
  - `TestOpenCmd_WithPicker_RecvFailedPrecondition_FallsBack` in `open_test.go` documents the correct
    fallback to `openAllAttached` when the picker's `stream.Recv()` returns `FailedPrecondition`
    (e.g. when `/dev/tty` is unavailable inside the picker handler).

## 3. Verification (cmd/swm)

- [x] 3.1 Run `task test` across all affected modules (`cmd/swm`, `plugins/session-tmux`) and confirm all tests pass
  - All modules pass: `cmd/swm`, `plugins/session-tmux`, `plugins/picker-fzf`, and all others.
- [x] 3.2 Run `swm workspace open --story <name>` end-to-end and confirm the picker opens and a pane group is created successfully
  - End-to-end run revealed two additional bugs fixed as follow-on stacked branches:
    - Silent exit due to `code_root=~/code` (tilde never expanded) → fixed in config.Load() with expandTilde()
    - Added `--log-level` flag + slog.Debug instrumentation to make silent failures visible
  - After those fixes, the picker opened with candidates, worktree was created, and OpenPaneGroup
    succeeded. SwitchTo then failed with "not a terminal" (plugin subprocess has no TTY) — this
    is a separate architectural issue tracked in a new change.
