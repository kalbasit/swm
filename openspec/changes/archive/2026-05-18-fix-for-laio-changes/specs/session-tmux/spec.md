## MODIFIED Requirements

### Requirement: paneGroupCommand exposes tmux_socket template variable
`session-tmux` SHALL substitute `{{tmux_socket}}` in `pane_group_command` with the absolute path of the story's tmux socket (the same value as `workspace_id` in the request).

#### Scenario: tmux_socket is substituted
- **WHEN** `config.toml` has `pane_group_command = "laio start --file {{worktree_path}}/.swm/laio.yaml --tmux-socket {{tmux_socket}} --replace-current-session --skip-attach"` and `OpenPaneGroup` is called with `workspace_id = /run/user/1000/swm/tmux/feat-x.sock`
- **THEN** the session's first window runs `laio start --file <worktree_path>/.swm/laio.yaml --tmux-socket /run/user/1000/swm/tmux/feat-x.sock --replace-current-session --skip-attach` with both `{{worktree_path}}` and `{{tmux_socket}}` expanded

#### Scenario: tmux_socket absent when no pane_group_command configured
- **WHEN** no `pane_group_command` is set in `config.toml`
- **THEN** the default layout is used and `{{tmux_socket}}` substitution does not occur

### Requirement: OpenPaneGroup in existing workspace
`session-tmux` SHALL implement `Session.OpenPaneGroup({story_name, project_id, worktree_path})` by creating a new tmux session for the project within the story's socket (if it doesn't exist). The initial working directory SHALL be `worktree_path`. The default layout SHALL run `$EDITOR` (or `vim` if unset) in the first window and a shell in the second, unless `pane_group_command` is configured.

#### Scenario: Default layout
- **WHEN** `OpenPaneGroup` is called and no `pane_group_command` is configured
- **THEN** a tmux session is created with a first window running `$EDITOR` and a second window running `$SHELL` in `worktree_path`

#### Scenario: Custom pane_group_command
- **WHEN** `config.toml` has `pane_group_command = "laio start --file {{worktree_path}}/.swm/laio.yaml --tmux-socket {{tmux_socket}} --replace-current-session --skip-attach"` and `OpenPaneGroup` is called
- **THEN** the session's first window runs `laio start --file <worktree_path>/.swm/laio.yaml --tmux-socket <tmux_socket> --replace-current-session --skip-attach` with `{{worktree_path}}` and `{{tmux_socket}}` both expanded

#### Scenario: Idempotent for existing session
- **WHEN** `OpenPaneGroup` is called for a project whose session already exists on the socket
- **THEN** the existing session is reused and no new session is created
