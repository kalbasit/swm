## MODIFIED Requirements

### Requirement: swm story create
`swm story create <name> [--branch <branch>]` SHALL create a new story JSON file via the story store. If `--branch` is omitted, the branch name SHALL be derived by evaluating the `branch_name_template` value from `config.toml` `[story]` section against the story name. The template MUST use Go `text/template` syntax with a single data variable `.Name` (the story name). When no template is configured, the default template `feat/{{.Name}}` SHALL be used, producing `feat/<name>` (backward-compatible). The command SHALL error if a story with the same name already exists. No worktrees are created (lazy). No plugins are invoked.

If the configured template is syntactically invalid, the command SHALL return an error before any hook or store operation. If the evaluated template produces an empty string, the command SHALL return an error.

Before creating the story JSON the command SHALL run `hookexec.Run` for event `pre-story-create` with the story name set. If any `pre-story-create` hook returns non-zero the command SHALL abort and exit non-zero. After creating the story JSON the command SHALL run `hookexec.Run` for event `post-story-create`; failures are logged but do not affect the exit code.

#### Scenario: Basic story creation
- **WHEN** `swm story create feat-x` is run with no config template set
- **THEN** `$XDG_DATA_HOME/swm/stories/feat-x.json` is created with `name="feat-x"`, `branch_name="feat/feat-x"`, and the command exits 0

#### Scenario: Custom branch name via --branch flag
- **WHEN** `swm story create JIRA-42 --branch fix/JIRA-42-crash` is run
- **THEN** the story JSON has `branch_name="fix/JIRA-42-crash"`

#### Scenario: Template from config used when --branch omitted
- **WHEN** `config.toml` sets `branch_name_template = "fix/{{.Name}}"` and `swm story create my-bug` is run without `--branch`
- **THEN** the story JSON has `branch_name="fix/my-bug"` and the command exits 0

#### Scenario: --branch flag overrides configured template
- **WHEN** `config.toml` sets `branch_name_template = "fix/{{.Name}}"` and `swm story create my-bug --branch custom/branch` is run
- **THEN** the story JSON has `branch_name="custom/branch"` (template is not evaluated)

#### Scenario: Invalid template in config yields error
- **WHEN** `config.toml` sets `branch_name_template = "{{.Name"` (unclosed action) and `swm story create feat-x` is run
- **THEN** the command exits non-zero with an error message referencing the template parse failure, and no story JSON is created

#### Scenario: Template evaluating to empty string yields error
- **WHEN** `config.toml` sets `branch_name_template = ""` and `swm story create feat-x` is run
- **THEN** the command exits non-zero with an error message indicating the branch name cannot be empty, and no story JSON is created

#### Scenario: Duplicate name
- **WHEN** `swm story create feat-x` is run and a story named `feat-x` already exists
- **THEN** the command exits non-zero with an appropriate error

#### Scenario: pre-story-create hook aborts creation
- **WHEN** a `pre-story-create` hook exits non-zero
- **THEN** the story JSON is NOT created and the command exits non-zero

#### Scenario: post-story-create hook fails — logged, command succeeds
- **WHEN** all `pre-story-create` hooks pass and a `post-story-create` hook exits non-zero
- **THEN** the story JSON is created, the failure is logged, and the command exits 0

## ADDED Requirements

### Requirement: branch_name_template config field
The `[story]` TOML section of `$XDG_CONFIG_HOME/swm/config.toml` SHALL support an optional `branch_name_template` string field. When present and non-empty it MUST be a valid Go `text/template` string. When absent or empty the host SHALL behave as if `"feat/{{.Name}}"` were specified. The template is evaluated with a single data struct exposing `.Name` (the story name string).

#### Scenario: Config with no story section uses default template
- **WHEN** `config.toml` contains no `[story]` section
- **THEN** the default template `feat/{{.Name}}` is used for branch name derivation

#### Scenario: Config with branch_name_template overrides default
- **WHEN** `config.toml` sets `[story] branch_name_template = "wael/{{.Name}}"`
- **THEN** `swm story create foo` produces a story with `branch_name="wael/foo"`

#### Scenario: Malformed template detected at story create time
- **WHEN** `config.toml` sets `branch_name_template = "{{.Name"` (parse error)
- **THEN** `swm story create` returns a non-zero exit code with a descriptive error before running any hooks or writing any files
