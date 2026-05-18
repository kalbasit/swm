## MODIFIED Requirements

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

### Requirement: OpenPaneGroup in existing workspace
`session-tmux` SHALL implement `Session.OpenPaneGroup({story_name, project_id, worktree_path})` by creating a new tmux session for the project within the story's socket (if it doesn't exist). The initial working directory SHALL be `worktree_path`. The default layout SHALL run `$EDITOR` (or `vim` if unset) in the first window and a shell in the second, unless `pane_group_command` is configured.

#### Scenario: Default layout
- **WHEN** `OpenPaneGroup` is called and no `pane_group_command` is configured
- **THEN** a tmux session is created with a first window running `$EDITOR` and a second window running `$SHELL` in `worktree_path`

#### Scenario: Custom pane_group_command
- **WHEN** `config.toml` has `pane_group_command = "laio start --file {{worktree_path}}/.swm/laio.yaml --socket {{tmux_socket}} --skip-attach"` and `OpenPaneGroup` is called
- **THEN** the session's first window runs `laio start --file <worktree_path>/.swm/laio.yaml --socket <tmux_socket> --skip-attach` with `{{worktree_path}}` and `{{tmux_socket}}` both expanded

#### Scenario: Idempotent for existing session
- **WHEN** `OpenPaneGroup` is called for a project whose session already exists on the socket
- **THEN** the existing session is reused and no new session is created
