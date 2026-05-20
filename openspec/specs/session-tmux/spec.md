## Purpose

The `session-tmux` plugin manages per-story tmux servers for swm. Each workspace (story) gets a dedicated tmux socket, and each pane group (project worktree) gets a named session within that socket. The plugin handles workspace lifecycle (create, attach, close), session navigation (SwitchTo), context detection (IsInsideWorkspace, CurrentContext), and optional per-session layout via `pane_group_command`.
## Requirements
### Requirement: Socket-per-workspace model
`session-tmux` SHALL map each swm workspace to a dedicated tmux server socket at `$XDG_RUNTIME_DIR/swm/tmux/<story-name>.sock`. Each pane group within a workspace SHALL map to a tmux session (named by sanitizing the full canonical path `host/seg1/.../segN` to be tmux-safe — replacing `.` with `•` (U+2022) and `:` with `：` (U+FF1A), e.g., `github•com/kalbasit/swm` for `github.com/kalbasit/swm`) within that socket. This preserves the v1 tmux isolation model while preventing collisions between same-named repos from different forges or orgs.

#### Scenario: Workspace socket path
- **WHEN** `OpenWorkspace({story_name: "feat-x", ...})` is called
- **THEN** the tmux server is started (if not running) on socket `$XDG_RUNTIME_DIR/swm/tmux/feat-x.sock`

#### Scenario: Pane group session name
- **WHEN** `OpenPaneGroup({story_name: "feat-x", project_id: {host: "github.com", segments: ["kalbasit", "swm"]}, ...})` is called
- **THEN** a tmux session named `github•com/kalbasit/swm` is created within the `feat-x.sock` server

#### Scenario: Session name collision prevention
- **WHEN** `OpenPaneGroup` is called for two projects with the same repo name but different orgs — `{host: "github.com", segments: ["org-a", "utils"]}` and `{host: "github.com", segments: ["org-b", "utils"]}` — within the same workspace
- **THEN** two distinct sessions `github•com/org-a/utils` and `github•com/org-b/utils` are created

### Requirement: OpenWorkspace creates and attaches
`session-tmux` SHALL implement `Session.OpenWorkspace({story_name, worktree_paths})` by starting the tmux server socket if it does not exist. When the server is started, a single bootstrap session named after the story SHALL be created to keep the server alive (tmux's `exit-empty on` default exits the server when there are no sessions). Project sessions SHALL NOT be pre-created; they are created lazily by `OpenPaneGroup` so that `pane_group_command` is applied to each one individually. If the socket already exists, the call is idempotent.

#### Scenario: New workspace
- **WHEN** `OpenWorkspace` is called for a story with no existing socket
- **THEN** a new tmux server is started on the story's socket, a single bootstrap session named after the story is created, and `Workspace` is returned

#### Scenario: Existing workspace
- **WHEN** `OpenWorkspace` is called and the story's socket already has a running server
- **THEN** the call completes without creating duplicate sessions

#### Scenario: Returns Workspace proto
- **WHEN** `OpenWorkspace` completes successfully
- **THEN** a `Workspace` message with `name = story_name` and `id = socket_path` is returned

### Requirement: CloseWorkspace terminates server
`session-tmux` SHALL implement `Session.CloseWorkspace({story_name})` by sending `tmux -S <socket> kill-server`. The socket file SHALL be cleaned up.

#### Scenario: Close running workspace
- **WHEN** `CloseWorkspace({story_name: "feat-x"})` is called and the socket is active
- **THEN** `tmux kill-server` is run on the socket and the socket file is removed

#### Scenario: Close non-existent workspace
- **WHEN** `CloseWorkspace` is called for a story with no socket file
- **THEN** the call succeeds (idempotent) with no error

### Requirement: ListWorkspaces streams active sockets
`session-tmux` SHALL implement `Session.ListWorkspaces()` by scanning `$XDG_RUNTIME_DIR/swm/tmux/` for socket files, probing each with `tmux -S <socket> list-sessions -F ""` to confirm the server is alive, and streaming one `Workspace` message per live socket.

#### Scenario: Multiple active workspaces
- **WHEN** `ListWorkspaces()` is called and two story sockets are live
- **THEN** two `Workspace` messages are streamed

#### Scenario: Stale socket files ignored
- **WHEN** a socket file exists but the tmux server is no longer running
- **THEN** that socket is excluded from the streamed results

### Requirement: paneGroupCommand exposes tmux_socket template variable
`session-tmux` SHALL substitute `{{tmux_socket}}` in `pane_group_command` with the absolute path of the story's tmux socket (the same value as `workspace_id` in the request).

#### Scenario: tmux_socket is substituted
- **WHEN** `config.toml` has `pane_group_command = "laio start --file '{{worktree_path}}/.swm/laio.yaml' --tmux-socket '{{tmux_socket}}' --replace-current-session --skip-attach"` and `OpenPaneGroup` is called with `workspace_id = /run/user/1000/swm/tmux/feat-x.sock`
- **THEN** the session's first window runs `laio start --file '<worktree_path>/.swm/laio.yaml' --tmux-socket '/run/user/1000/swm/tmux/feat-x.sock' --replace-current-session --skip-attach` with both `{{worktree_path}}` and `{{tmux_socket}}` expanded

#### Scenario: tmux_socket absent when no pane_group_command configured
- **WHEN** no `pane_group_command` is set in `config.toml`
- **THEN** the default layout is used and `{{tmux_socket}}` substitution does not occur

### Requirement: OpenPaneGroup in existing workspace
`session-tmux` SHALL implement `Session.OpenPaneGroup({story_name, project_id, worktree_path})` by creating a new tmux session for the project within the story's socket (if it doesn't exist). The initial working directory SHALL be `worktree_path`. The default layout SHALL run `$EDITOR` (or `vim` if unset) in the first window and a shell in the second, unless `pane_group_command` is configured.

When `pane_group_command` is configured, `session-tmux` SHALL validate that the first token of the command resolves to an executable via PATH lookup before creating the tmux session. If the binary is not found, `OpenPaneGroup` SHALL return a `FailedPrecondition` error naming the missing binary — no tmux session SHALL be created.

#### Scenario: Default layout
- **WHEN** `OpenPaneGroup` is called and no `pane_group_command` is configured
- **THEN** a tmux session is created with a first window running `$EDITOR` and a second window running `$SHELL` in `worktree_path`

#### Scenario: Custom pane_group_command
- **WHEN** `config.toml` has `pane_group_command = "laio start --file '{{worktree_path}}/.swm/laio.yaml' --tmux-socket '{{tmux_socket}}' --replace-current-session --skip-attach"` and `OpenPaneGroup` is called
- **THEN** the session's first window runs `laio start --file '<worktree_path>/.swm/laio.yaml' --tmux-socket '<tmux_socket>' --replace-current-session --skip-attach` with `{{worktree_path}}` and `{{tmux_socket}}` both expanded

#### Scenario: Idempotent for existing session
- **WHEN** `OpenPaneGroup` is called for a project whose session already exists on the socket
- **THEN** the existing session is reused and no new session is created

#### Scenario: pane_group_command binary not found
- **WHEN** `config.toml` has `pane_group_command = "laio start ..."` and `laio` is not present in PATH
- **THEN** `OpenPaneGroup` returns a `FailedPrecondition` error with a message identifying the missing binary (e.g. `pane_group_command binary "laio" not found in PATH`)
- **AND** no tmux session is created

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

### Requirement: IsInsideWorkspace detection
`session-tmux` SHALL implement `Session.IsInsideWorkspace()` by checking whether `$TMUX` is set and the socket path matches `$XDG_RUNTIME_DIR/swm/tmux/<story>.sock` for any known story. Returns `BoolValue{value: true}` if inside a swm-managed workspace.

#### Scenario: Inside swm tmux workspace
- **WHEN** `IsInsideWorkspace()` is called with `$TMUX` pointing to a swm workspace socket
- **THEN** `BoolValue{value: true}` is returned

#### Scenario: Outside any tmux
- **WHEN** `IsInsideWorkspace()` is called with `$TMUX` unset
- **THEN** `BoolValue{value: false}` is returned

### Requirement: CurrentContext returns active workspace and pane group
`session-tmux` SHALL implement `Session.CurrentContext()` by reading `$TMUX` for the socket path and `$TMUX_PANE`/`tmux display-message` for the active session name. Returns a `CurrentContextResponse` with `workspace_id` and `pane_group_id`.

#### Scenario: Inside a swm workspace
- **WHEN** `CurrentContext()` is called from within a swm-managed tmux session
- **THEN** `CurrentContextResponse` is returned with the story name derived from the socket path and the pane group name from the active session

### Requirement: Environment isolation at workspace launch

Before launching the tmux server process, the session plugin MUST explicitly construct the child process environment using a denylist approach: start from the inherited `os.Environ()` and strip all plugin-internal variables. The resulting environment MUST NOT contain `SWM_HOST_SOCKET`, `SWM_LOG_LEVEL`, or `SWM_PLUGIN_MAGIC_COOKIE`.

The canonical environment variable ownership model for a user session:

| Variable | Present in tmux session |
|---|:---:|
| `SWM_HOST_SOCKET` | no — stripped |
| `SWM_LOG_LEVEL` | no — stripped |
| `SWM_PLUGIN_MAGIC_COOKIE` | no — stripped |
| `SWM_STORY` | yes — set by session plugin at workspace open |
| All other user env vars | yes — inherited unchanged |

#### Scenario: Plugin-internal vars absent from new tmux window
- **WHEN** a workspace is opened via `OpenWorkspace` and a new shell is spawned in a tmux window
- **THEN** `SWM_HOST_SOCKET` is absent from the shell's environment
- **AND** `SWM_LOG_LEVEL` is absent from the shell's environment
- **AND** `SWM_PLUGIN_MAGIC_COOKIE` is absent from the shell's environment

#### Scenario: User environment preserved in tmux session
- **WHEN** a workspace is opened and the user had `HOME`, `PATH`, and arbitrary user-defined vars set before invoking swm
- **THEN** those variables are present and unchanged in the tmux session's shell environment

#### Scenario: SWM_STORY present in tmux session
- **WHEN** a workspace is opened for story `<story-name>`
- **THEN** `SWM_STORY` is set to `<story-name>` in the tmux session environment

