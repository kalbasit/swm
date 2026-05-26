## MODIFIED Requirements

### Requirement: OpenPaneGroup in existing workspace
`session-tmux` SHALL implement `Session.OpenPaneGroup({story_name, project_id, worktree_path})` by creating a new tmux session for the project within the story's socket (if it doesn't exist). The initial working directory SHALL be `worktree_path`.

`OpenPaneGroup` SHALL resolve the layout using the following priority order (first match wins):

1. If `pane_group_command` is set in `config.toml`: run that command (existing behavior). A warning SHALL be logged if a layout config file also exists at either tier.
2. If `<worktree_path>/.swm/session-tmux.toml` exists: apply the per-repo layout (see `session-tmux-layout` spec).
3. If `$XDG_CONFIG_HOME/swm/session-tmux.toml` exists: apply the global layout (see `session-tmux-layout` spec).
4. Default: run `$EDITOR` (or `vim` if unset) in the first window and a shell in the second.

When `pane_group_command` is configured, `session-tmux` SHALL validate that the first token of the command resolves to an executable via PATH lookup before creating the tmux session. If the binary is not found, `OpenPaneGroup` SHALL return a `FailedPrecondition` error naming the missing binary — no tmux session SHALL be created.

#### Scenario: Default layout
- **WHEN** no `pane_group_command` is configured and no layout config exists at either tier
- **THEN** the tmux session is created with two windows: the first running `$EDITOR` (or `vim`), the second running a shell

#### Scenario: Custom pane_group_command
- **WHEN** `config.toml` has `pane_group_command = "laio start --file '{{worktree_path}}/.swm/laio.yaml' --tmux-socket '{{tmux_socket}}' --replace-current-session --skip-attach"` and `OpenPaneGroup` is called
- **THEN** the session's first window runs `laio start --file '<worktree_path>/.swm/laio.yaml' --tmux-socket '<tmux_socket>' --replace-current-session --skip-attach` with `{{worktree_path}}` and `{{tmux_socket}}` both expanded

#### Scenario: Per-repo layout config applied
- **WHEN** `<worktree_path>/.swm/session-tmux.toml` exists and `pane_group_command` is not set
- **THEN** the layout defined in that file is applied to the newly created tmux session

#### Scenario: Global layout config applied
- **WHEN** `$XDG_CONFIG_HOME/swm/session-tmux.toml` exists, `pane_group_command` is not set, and no per-repo config exists
- **THEN** the layout defined in the global config is applied to the newly created tmux session

#### Scenario: Per-repo layout wins over global
- **WHEN** both `<worktree_path>/.swm/session-tmux.toml` and `$XDG_CONFIG_HOME/swm/session-tmux.toml` exist and `pane_group_command` is not set
- **THEN** only the per-repo config is applied

#### Scenario: pane_group_command wins when layout config also present
- **WHEN** `pane_group_command` is set and `<worktree_path>/.swm/session-tmux.toml` also exists
- **THEN** `pane_group_command` is used, a warning is logged naming the ignored layout file, and the layout config is not read

#### Scenario: Idempotent for existing session
- **WHEN** `OpenPaneGroup` is called for a project whose session already exists on the socket
- **THEN** the existing session is reused and no new session is created

#### Scenario: pane_group_command binary not found
- **WHEN** `pane_group_command` is set to a command whose binary does not exist in `PATH`
- **THEN** `OpenPaneGroup` returns a `FailedPrecondition` error naming the missing binary, and no tmux session is created
