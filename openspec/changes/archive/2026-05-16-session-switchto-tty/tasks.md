## 1. Proto: extend SwitchToResponse (proto)

- [x] 1.1 In `proto/swm/plugin/v1/session.proto`, add a new `SwitchToResponse` message with `repeated string exec_argv = 1;`
- [x] 1.2 In the same file, change `rpc SwitchTo(SwitchToRequest) returns (Empty)` to `returns (SwitchToResponse)`
- [x] 1.3 Run `task proto:gen` to regenerate `session.pb.go` and `session_grpc.pb.go`
- [x] 1.4 Verify `buf build` and `task proto:lint` pass with zero issues

## 2. Plugin: update SwitchTo to return exec_argv (plugins/session-tmux)

- [x] 2.1 In `plugins/session-tmux/internal/session/tmux_test.go`, add `TestSwitchTo_OutsideTmux_ReturnsExecArgv`: verify that when `$TMUX` is unset, `SwitchTo` returns a non-empty `exec_argv` containing the tmux attach-session command and does NOT call tmux itself
- [x] 2.2 In `plugins/session-tmux/internal/session/tmux_test.go`, add `TestSwitchTo_InsideTmux_CallsSwitchClient`: verify that when `$TMUX` is set, `SwitchTo` calls `switch-client` and returns empty `exec_argv`
- [x] 2.3 In `plugins/session-tmux/internal/session/tmux.go`, update `SwitchTo` signature to return `(*pluginv1.SwitchToResponse, error)`
- [x] 2.4 When `os.Getenv("TMUX") == ""`: return `&pluginv1.SwitchToResponse{ExecArgv: []string{t.tmuxBin, "-S", sock, "attach-session", "-t", target}}` without running tmux
- [x] 2.5 When `os.Getenv("TMUX") != ""`: call `switch-client` (existing logic) and return `&pluginv1.SwitchToResponse{}` on success
- [x] 2.6 Run `task session-tmux:test` and confirm all tests pass

## 3. Host: exec the returned command (cmd/swm)

- [x] 3.1 In `cmd/swm/internal/cli/workspace/open_test.go`, update `stubSess.SwitchTo` stub to return `*pluginv1.SwitchToResponse` (compile fix after proto change)
- [x] 3.2 In `cmd/swm/internal/cli/workspace/open_test.go`, add `TestOpenCmd_SwitchTo_ExecArgv_IsExeced`: mock the exec function and verify it is called with the `exec_argv` returned by the stub session
- [x] 3.3 In `cmd/swm/internal/cli/workspace/open.go`, introduce an `execFunc` variable (default `syscall.Exec`) to allow test injection
- [x] 3.4 In `openWithPicker`: after `post-workspace-open` hooks, call `SwitchTo`; if `exec_argv` is non-empty, call `execFunc(exec_argv[0], exec_argv, os.Environ())`
- [x] 3.5 In `openAllAttached`: no `SwitchTo` call needed — this path never called it and we have no pane_group_id here; `execFn` was not added to avoid linter unused-param warning
- [x] 3.6 Run `task swm:test` and confirm all tests pass

## 4. Verification

- [x] 4.1 Run `task fmt && task lint && task test` across the full repo and confirm zero issues
- [ ] 4.2 Build and install `nix profile install .#swm-full`
- [ ] 4.3 Run `swm workspace open --story <name>` end-to-end and confirm fzf opens, a project can be selected, and the terminal attaches to the tmux session
