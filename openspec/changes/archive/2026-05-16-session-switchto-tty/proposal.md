## Why

`swm workspace open` fails with "open terminal failed: not a terminal" when
the user is not already inside a tmux session. The session-tmux plugin runs
`tmux attach-session` from within its gRPC subprocess, which has pipes — not
a TTY. Only the host process (swm itself) has access to the terminal.

## What Changes

- Extend `SwitchToResponse` in `proto/swm/plugin/v1/session.proto` to carry
  an optional `repeated string exec_argv` field (**BREAKING** — proto change)
- `rpc SwitchTo` return type changes from `Empty` to `SwitchToResponse`
- Plugin: when `$TMUX == ""`, return `exec_argv = ["tmux", "-S", <sock>,
  "attach-session", "-t", <target>]` without running attach-session itself
- Plugin: when `$TMUX != ""`, call `switch-client` as before and return
  empty `exec_argv`
- Host: after `SwitchTo` returns, if `exec_argv` is non-empty, call
  `syscall.Exec` to replace itself with the specified command, inheriting
  the terminal

## Capabilities

### New Capabilities
_(none)_

### Modified Capabilities
- `workflow-commands`: `swm workspace open` now execs tmux attach-session
  from the host process rather than through the plugin, so the terminal is
  inherited correctly

## Non-goals

- Changing how `switch-client` works (already correct for in-tmux case)
- Supporting non-tmux session plugins differently (the `exec_argv` field is
  generic enough; other plugins may leave it empty)
- Handling the case where the user cancels after the pane group is created

## Impact

- `proto/swm/plugin/v1/session.proto`: new `SwitchToResponse` message;
  `rpc SwitchTo` return changes from `Empty` to `SwitchToResponse`
- All generated files under `proto/swm/plugin/v1/` must be regenerated
  (`task proto:gen`)
- `plugins/session-tmux/internal/session/tmux.go`: `SwitchTo` signature and
  logic updated
- `cmd/swm/internal/cli/workspace/open.go`: `syscall.Exec` call added after
  `SwitchTo`
- `cmd/swm/internal/cli/workspace/open_test.go`: stub updated for new return
  type
- `plugins/session-tmux/internal/session/tmux_test.go`: tests for new
  `SwitchTo` behaviour
