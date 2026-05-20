## MODIFIED Requirements

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
