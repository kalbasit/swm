## ADDED Requirements

### Requirement: session-tmux README documents template variables

The `plugins/session-tmux/README.md` SHALL include a template-variable reference table listing
all variables available for substitution in `pane_group_command`.

#### Scenario: Template-variable table is present and complete
- **WHEN** a user reads `plugins/session-tmux/README.md`
- **THEN** it SHALL contain a table listing `{{worktree_path}}`, `{{story_name}}`,
  `{{project_id}}`, and `{{tmux_socket}}` with a description of each

---

### Requirement: session-tmux README documents laio integration

The `plugins/session-tmux/README.md` SHALL include a "Laio integration" section that shows
how to wire [laio](https://github.com/stephane-klein/laio) into `pane_group_command`.

#### Scenario: Per-project laio config example is present
- **WHEN** a user reads the laio integration section
- **THEN** it SHALL contain a `config.toml` snippet using
  `pane_group_command = "laio start --file {{worktree_path}}/.swm/laio.yaml --socket {{tmux_socket}} --skip-attach"`

#### Scenario: Global laio config example is present
- **WHEN** a user reads the laio integration section
- **THEN** it SHALL contain a `config.toml` snippet using a fixed `--file` path with
  `--var path={{worktree_path}}` and a corresponding `laio.yaml` fragment showing
  `path: "{{ path }}"`

#### Scenario: --skip-attach requirement is explained
- **WHEN** a user reads the laio integration section
- **THEN** it SHALL explain that `--skip-attach` is required because laio runs inside an
  already-attached session

---

### Requirement: session-tmux plugin ships a sample laio.yaml

`plugins/session-tmux/examples/laio.yaml` SHALL exist and contain a working multi-window
laio configuration that is compatible with swm's socket model.

#### Scenario: Sample laio.yaml file exists
- **WHEN** the repository is inspected
- **THEN** `plugins/session-tmux/examples/laio.yaml` SHALL exist

#### Scenario: Sample laio.yaml uses the path variable
- **WHEN** a user reads `plugins/session-tmux/examples/laio.yaml`
- **THEN** it SHALL use `path: "{{ path }}"` to accept the worktree path via `--var path=<value>`
