### Requirement: swm story create
`swm story create <name> [--branch <branch>]` SHALL create a new story JSON file via the story store. If `--branch` is omitted, the branch name SHALL default to `feat/<name>`. The command SHALL error if a story with the same name already exists. No worktrees are created (lazy). No plugins are invoked.

Before creating the story JSON the command SHALL run `hookexec.Run` for event `pre-story-create` with the story name set. If any `pre-story-create` hook returns non-zero the command SHALL abort and exit non-zero. After creating the story JSON the command SHALL run `hookexec.Run` for event `post-story-create`; failures are logged but do not affect the exit code.

#### Scenario: Basic story creation
- **WHEN** `swm story create feat-x` is run
- **THEN** `$XDG_DATA_HOME/swm/stories/feat-x.json` is created with `name="feat-x"`, `branch_name="feat/feat-x"`, and the command exits 0

#### Scenario: Custom branch name
- **WHEN** `swm story create JIRA-42 --branch fix/JIRA-42-crash` is run
- **THEN** the story JSON has `branch_name="fix/JIRA-42-crash"`

#### Scenario: Duplicate name
- **WHEN** `swm story create feat-x` is run and `feat-x.json` already exists
- **THEN** the command exits non-zero with an error message referencing the story name

#### Scenario: pre-story-create hook aborts creation
- **WHEN** a `pre-story-create` hook exits non-zero
- **THEN** the story JSON is NOT created and the command exits non-zero

#### Scenario: post-story-create hook fails — logged, command succeeds
- **WHEN** all `pre-story-create` hooks pass and a `post-story-create` hook exits non-zero
- **THEN** the story JSON is created, the failure is logged, and the command exits 0

### Requirement: swm story remove
`swm story remove <name> [--force]` SHALL remove a story and all its worktrees. Without `--force`, a confirmation prompt MUST be shown listing all attached projects. The removal sequence SHALL be:
1. Run `pre-story-remove` hooks; abort if any fail.
2. For each attached project: run `pre-worktree-remove` hooks, call `vcs.RemoveWorktree`, run `post-worktree-remove` hooks.
3. Call `session.CloseWorkspace` if a workspace exists.
4. Delete the story JSON.
5. Run `post-story-remove` hooks (failures logged, not fatal).

If any step fails the remaining steps SHALL still be attempted (best-effort cleanup) and a summary of failures is printed.

#### Scenario: With --force skips prompt
- **WHEN** `swm story remove feat-x --force` is run
- **THEN** the removal proceeds without any confirmation prompt

#### Scenario: Without --force shows prompt
- **WHEN** `swm story remove feat-x` is run interactively
- **THEN** a prompt listing the story's projects is shown; entering `y` proceeds, `n` aborts

#### Scenario: Unknown story
- **WHEN** `swm story remove nonexistent` is run
- **THEN** the command exits non-zero with an error indicating the story was not found

#### Scenario: Story with no projects
- **WHEN** `swm story remove feat-x --force` is run and `feat-x` has no attached projects
- **THEN** no VCS calls are made and the story JSON is deleted

#### Scenario: pre-story-remove hook aborts removal
- **WHEN** a `pre-story-remove` hook exits non-zero
- **THEN** the removal is aborted and the story JSON is NOT deleted

### Requirement: swm clone
`swm clone <url>` SHALL clone a repository to its canonical path. The flow:
1. Run `pre-clone` hooks; abort if any fail.
2. Call `vcs.ParseRemoteURL(url)` to get `ProjectID`.
3. Compose canonical path from code root + project ID.
4. If canonical path already has `.git`, print "already cloned" and exit 0.
5. Call `vcs.Clone(url, canonical_path)`.
6. Run `post-clone` hooks (failures logged, not fatal).

The repository is NOT attached to any story.

#### Scenario: Successful clone
- **WHEN** `swm clone git@github.com:kalbasit/swm.git` is run
- **THEN** the repository is cloned to `$CODE_ROOT/repositories/github.com/kalbasit/swm/` and the command exits 0

#### Scenario: Already cloned
- **WHEN** `swm clone git@github.com:kalbasit/swm.git` is run and the canonical path already exists with a `.git` directory
- **THEN** the command prints "already cloned at <path>" and exits 0 without calling `vcs.Clone`

#### Scenario: Clone failure
- **WHEN** `vcs.Clone` returns an error (e.g., network error, auth failure)
- **THEN** the command exits non-zero with the error message from the VCS plugin

#### Scenario: pre-clone hook aborts clone
- **WHEN** a `pre-clone` hook exits non-zero
- **THEN** `vcs.Clone` is NOT called and the command exits non-zero

### Requirement: swm workspace open
`swm workspace open [--story <name>]` SHALL open (or switch to) the tmux workspace for a story. The flow depends on whether a picker plugin is configured:

Before opening the workspace the command SHALL run `hookexec.Run` for event `pre-workspace-open` with the story name set. If any `pre-workspace-open` hook returns non-zero the command SHALL abort. After the workspace is open the command SHALL run `hookexec.Run` for event `post-workspace-open`; failures are logged but do not affect the exit code.

**With picker configured:**
1. Run `pre-workspace-open` hooks; abort if any fail.
2. Resolve story from `--story` flag, `$SWM_STORY` env var, or `_default`.
3. Build a candidate list: all projects already attached to the story plus all repositories discovered under `$CODE_ROOT/repositories/` via `host.ListProjects`.
4. Stream candidates to `picker.Pick`; each candidate's `display` is its project ID string (e.g. `github.com/kalbasit/swm`) and `key` is the same string.
5. Receive the selected project ID from the picker.
6. If the selected project is NOT already attached to the story: call `vcs.CreateWorktree` for that project and attach it to the story in the story store.
7. Call `session.OpenWorkspace` to ensure the workspace is active.
8. Call `session.OpenPaneGroup` with the derived worktree path for the selected project.
9. Run `post-workspace-open` hooks.

**Without picker configured (fallback):**
1. Run `pre-workspace-open` hooks; abort if any fail.
2. Resolve story.
3. Load all attached projects from the story store.
4. Call `session.OpenWorkspace({story_name, worktree_paths: {project_key: derived_path}})`.
5. If the workspace was already open, call `session.SwitchTo` for the first pane group.
6. Run `post-workspace-open` hooks.

#### Scenario: Interactive selection with picker — project already attached
- **WHEN** `swm workspace open --story feat-x` is run, a picker is configured, and the user selects a project already attached to `feat-x`
- **THEN** `picker.Pick` is called with all candidates, `session.OpenPaneGroup` is called with the selected project's worktree path, and no `vcs.CreateWorktree` call is made

#### Scenario: Interactive selection with picker — project not yet attached
- **WHEN** `swm workspace open --story feat-x` is run, a picker is configured, and the user selects a project NOT yet attached to `feat-x`
- **THEN** `vcs.CreateWorktree` is called for the selected project, the project is recorded in the story store, and `session.OpenPaneGroup` is called with the new worktree path

#### Scenario: Picker cancelled by user
- **WHEN** `swm workspace open --story feat-x` is run, a picker is configured, and the user cancels the picker (Escape / Ctrl-C)
- **THEN** the command exits 0 with no workspace changes and no error message to the user

#### Scenario: No picker configured — opens all attached projects
- **WHEN** `swm workspace open --story feat-x` is run and no picker plugin is configured
- **THEN** `session.OpenWorkspace` is called with all projects attached to `feat-x` (Phase 1 behaviour)

#### Scenario: Story from environment
- **WHEN** `swm workspace open` is run with `$SWM_STORY=feat-x` set
- **THEN** the workspace for `feat-x` is opened (same as `--story feat-x`)

#### Scenario: Default story
- **WHEN** `swm workspace open` is run with no `--story` flag and no `$SWM_STORY`
- **THEN** the workspace for the `_default` story is opened

#### Scenario: Story not found
- **WHEN** `swm workspace open --story nonexistent` is run
- **THEN** the command exits non-zero with an error indicating the story was not found

#### Scenario: Story with no projects and no picker
- **WHEN** `swm workspace open --story feat-x` is run, no picker is configured, and `feat-x` has no attached projects
- **THEN** `session.OpenWorkspace` is called with an empty `worktree_paths` map; the session plugin opens an empty workspace

#### Scenario: pre-workspace-open hook aborts open
- **WHEN** a `pre-workspace-open` hook exits non-zero
- **THEN** no workspace is opened and the command exits non-zero

### Requirement: swm story list
`swm story list` SHALL print all story names to stdout, one per line, in
lexical order. The command takes no arguments and no flags. On success it exits
zero. If the store cannot be read it exits non-zero and prints an error to
stderr.

#### Scenario: Single story (default only)
- **WHEN** `swm story list` is run and only the `_default` story exists
- **THEN** the command exits zero and prints exactly `_default` to stdout

#### Scenario: Multiple stories
- **WHEN** `swm story list` is run and stories `alpha`, `beta`, and `_default` exist
- **THEN** the command exits zero and prints the names in lexical order, one per line

#### Scenario: Store error
- **WHEN** `swm story list` is run and `Store.List` returns an error
- **THEN** the command exits non-zero and prints a human-readable error message

### Requirement: Config file resolution order
When `swm` starts it SHALL resolve the configuration file path using the following precedence (first match wins):
1. `$SWM_CONFIG` environment variable, if set and non-empty.
2. `$XDG_CONFIG_HOME/swm/config.toml` (where `$XDG_CONFIG_HOME` defaults to `~/.config` per the XDG Base Directory Specification).

If the resolved file does not exist, `swm` SHALL start with built-in defaults and SHALL NOT treat a missing file as an error.

#### Scenario: SWM_CONFIG env var overrides XDG default
- **WHEN** `$SWM_CONFIG` is set to `/custom/path/config.toml` and that file exists
- **THEN** `swm` loads config from `/custom/path/config.toml`, ignoring `$XDG_CONFIG_HOME/swm/config.toml`

#### Scenario: XDG default used when SWM_CONFIG is unset
- **WHEN** `$SWM_CONFIG` is unset and `$XDG_CONFIG_HOME/swm/config.toml` exists with `[plugins] session = "tmux"`
- **THEN** `swm` loads config from the XDG path and plugin commands succeed

#### Scenario: Missing config file falls back to defaults
- **WHEN** `$SWM_CONFIG` is unset and `$XDG_CONFIG_HOME/swm/config.toml` does not exist
- **THEN** `swm` starts with built-in defaults (code_root=~/code, default_story=_default, no plugins) and exits zero
