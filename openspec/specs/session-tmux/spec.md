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
`session-tmux` SHALL implement `Session.OpenWorkspace({story_name, worktree_paths})` by starting the tmux server socket if it does not exist, creating one session per project in `worktree_paths`, and attaching the current terminal to the workspace. If the socket already exists (workspace is already open), it SHALL attach to the existing workspace without recreating sessions.

#### Scenario: New workspace
- **WHEN** `OpenWorkspace` is called for a story with no existing socket
- **THEN** a new tmux server is started on the story's socket, one session is created per project, and the terminal is attached

#### Scenario: Existing workspace
- **WHEN** `OpenWorkspace` is called and the story's socket already has a running server
- **THEN** the terminal is attached to the existing workspace without creating duplicate sessions

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

### Requirement: OpenPaneGroup in existing workspace
`session-tmux` SHALL implement `Session.OpenPaneGroup({story_name, project_id, worktree_path})` by creating a new tmux session for the project within the story's socket (if it doesn't exist). The initial working directory SHALL be `worktree_path`. The default layout SHALL run `$EDITOR` (or `vim` if unset) in the first window and a shell in the second, unless `pane_group_command` is configured.

#### Scenario: Default layout
- **WHEN** `OpenPaneGroup` is called and no `pane_group_command` is configured
- **THEN** a tmux session is created with a first window running `$EDITOR` and a second window running `$SHELL` in `worktree_path`

#### Scenario: Custom pane_group_command
- **WHEN** `config.toml` has `pane_group_command = "laio start --config {{worktree_path}}/.swm/laio.yaml"` and `OpenPaneGroup` is called
- **THEN** the session's first window runs `laio start --config <worktree_path>/.swm/laio.yaml` with `{{worktree_path}}` expanded

#### Scenario: Idempotent for existing session
- **WHEN** `OpenPaneGroup` is called for a project that already has a session in the workspace
- **THEN** the existing session is returned without creating a duplicate

### Requirement: SwitchTo switches active pane group
`session-tmux` SHALL implement `Session.SwitchTo({story_name, project_id})` by running `tmux -S <socket> switch-client -t <session-name>` if already inside a tmux session, or `tmux -S <socket> attach-session -t <session-name>` otherwise.

#### Scenario: Switch when inside tmux
- **WHEN** `SwitchTo` is called from within an active tmux session
- **THEN** `tmux switch-client` is used to jump to the target session

#### Scenario: Attach when outside tmux
- **WHEN** `SwitchTo` is called from a terminal not inside any tmux session
- **THEN** `tmux attach-session` is used to connect

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
