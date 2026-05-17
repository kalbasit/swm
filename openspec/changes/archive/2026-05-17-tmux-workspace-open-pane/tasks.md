## 1. Proto — extend SwitchToRequest

- [x] 1.1 Add `close_origin_workspace_id string` (field 3) and `close_origin_pane_id string` (field 4) to `SwitchToRequest` in `proto/swm/plugin/v1/session.proto` (module: proto)
- [x] 1.2 Regenerate protobuf Go code via `task proto:gen` and commit the updated generated files (module: proto)
- [x] 1.3 Verify `cmd/swm` and `plugins/session-tmux` compile cleanly after proto regen (modules: cmd/swm, plugins/session-tmux)

## 2. session-tmux — implement close-origin in SwitchTo

- [x] 2.1 Update `SwitchTo` in `plugins/session-tmux/internal/session/tmux.go`: when `close_origin_pane_id` is non-empty, look up the socket path for `close_origin_workspace_id` in the workspace registry; return a `NotFound` gRPC error if the workspace is absent (module: plugins/session-tmux)
- [x] 2.2 After the switch (or after building `exec_argv`), run `tmux -S <origin_socket> kill-pane -t <close_origin_pane_id>`; swallow "no such pane" and "no such session" errors from tmux (module: plugins/session-tmux)
- [x] 2.3 Add table-driven tests in `tmux_test.go` covering: kill after in-place switch, kill on exec-argv path, pane-already-gone error swallowed, unknown origin workspace returns NotFound, empty `close_origin_pane_id` runs no kill (module: plugins/session-tmux)

## 3. cmd/swm — add --kill-pane flag to workspace open

- [x] 3.1 Add `--kill-pane` boolean flag to `cmd/swm/internal/cli/workspace/open.go` (module: cmd/swm)
- [x] 3.2 When `--kill-pane` is set and `os.Getenv("TMUX_PANE")` is non-empty, call `session.CurrentContext()` before calling `SwitchTo` to obtain the origin `workspace_id` (module: cmd/swm)
- [x] 3.3 Populate `SwitchToRequest.close_origin_workspace_id` and `SwitchToRequest.close_origin_pane_id` when both are available; omit them silently when `$TMUX_PANE` is empty or `CurrentContext()` fails (module: cmd/swm)
- [x] 3.4 Add tests in `open_test.go` covering: `--kill-pane` with `$TMUX_PANE` set (origin fields populated), `--kill-pane` with `$TMUX_PANE` unset (no-op), `--kill-pane` with `CurrentContext()` error (no-op), no `--kill-pane` flag (fields omitted) (module: cmd/swm)
