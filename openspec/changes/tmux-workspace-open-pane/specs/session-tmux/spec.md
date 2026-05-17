## MODIFIED Requirements

### Requirement: SwitchTo switches active pane group

`session-tmux` SHALL implement `Session.SwitchTo({workspace_id, pane_group_id, close_origin_workspace_id, close_origin_pane_id})` by running `tmux -S <target_socket> switch-client -t <session-name>` if already inside a tmux session, or building an `exec_argv` of `["tmux", "-S", "<target_socket>", "attach-session", "-t", "<session-name>"]` otherwise.

When `close_origin_pane_id` is non-empty, the plugin SHALL:
1. Look up the socket path for `close_origin_workspace_id` in its workspace registry.
2. If the workspace is not found, return an error.
3. After performing the switch (or building `exec_argv`), run `tmux -S <origin_socket> kill-pane -t <close_origin_pane_id>`.
4. Ignore any "no such pane" or "no such session" errors from `kill-pane`.

The kill MUST happen inside the RPC handler before the response is returned, so that it executes even when the host will subsequently call `syscall.Exec` with the returned `exec_argv`.

#### Scenario: Switch when inside tmux
- **WHEN** `SwitchTo` is called from within an active tmux session
- **THEN** `tmux switch-client` is used to jump to the target session

#### Scenario: Attach when outside tmux
- **WHEN** `SwitchTo` is called from a terminal not inside any tmux session
- **THEN** `exec_argv` of `["tmux", "-S", "<socket>", "attach-session", "-t", "<session>"]` is returned and the host execs it

#### Scenario: Kill origin pane after in-place switch
- **WHEN** `SwitchTo` is called from inside a tmux session with non-empty `close_origin_workspace_id` and `close_origin_pane_id`
- **THEN** after `tmux switch-client` completes, `tmux kill-pane -t <close_origin_pane_id>` is run on the origin socket, and `SwitchToResponse` is returned with empty `exec_argv`

#### Scenario: Kill origin pane on exec path
- **WHEN** `SwitchTo` is called from outside any tmux session with non-empty `close_origin_workspace_id` and `close_origin_pane_id`
- **THEN** `kill-pane` runs on the origin socket before the `exec_argv` response is returned

#### Scenario: Kill origin — pane already gone
- **WHEN** `SwitchTo` is called with `close_origin_pane_id` set but the pane no longer exists
- **THEN** the "no such pane" error from `tmux kill-pane` is ignored and `SwitchTo` returns success

#### Scenario: Kill origin — unknown workspace
- **WHEN** `SwitchTo` is called with a `close_origin_workspace_id` that is not present in the plugin's workspace registry
- **THEN** `SwitchTo` returns a `NotFound` gRPC error

#### Scenario: No kill when close_origin_pane_id is empty
- **WHEN** `SwitchTo` is called with empty `close_origin_pane_id`
- **THEN** no `kill-pane` command is run and behaviour is identical to the existing switch
