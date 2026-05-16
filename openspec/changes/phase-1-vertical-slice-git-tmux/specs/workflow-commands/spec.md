## ADDED Requirements

### Requirement: swm story create
`swm story create <name> [--branch <branch>]` SHALL create a new story JSON file via the story store. If `--branch` is omitted, the branch name SHALL default to `feat/<name>`. The command SHALL error if a story with the same name already exists. No worktrees are created (lazy). No plugins are invoked.

#### Scenario: Basic story creation
- **WHEN** `swm story create feat-x` is run
- **THEN** `$XDG_DATA_HOME/swm/stories/feat-x.json` is created with `name="feat-x"`, `branch_name="feat/feat-x"`, and the command exits 0

#### Scenario: Custom branch name
- **WHEN** `swm story create JIRA-42 --branch fix/JIRA-42-crash` is run
- **THEN** the story JSON has `branch_name="fix/JIRA-42-crash"`

#### Scenario: Duplicate name
- **WHEN** `swm story create feat-x` is run and `feat-x.json` already exists
- **THEN** the command exits non-zero with an error message referencing the story name

### Requirement: swm story remove
`swm story remove <name> [--force]` SHALL remove a story and all its worktrees. Without `--force`, a confirmation prompt MUST be shown listing all attached projects. The removal sequence SHALL be: (1) for each attached project call `vcs.RemoveWorktree`, (2) call `session.CloseWorkspace` if a workspace exists, (3) delete the story JSON. If any step fails the remaining steps SHALL still be attempted (best-effort cleanup) and a summary of failures is printed.

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

### Requirement: swm clone
`swm clone <url>` SHALL clone a repository to its canonical path. The flow: (1) call `vcs.ParseRemoteURL(url)` to get `ProjectID`, (2) compose canonical path from code root + project ID, (3) if canonical path already has `.git`, print "already cloned" and exit 0, (4) call `vcs.Clone(url, canonical_path)`. The repository is NOT attached to any story.

#### Scenario: Successful clone
- **WHEN** `swm clone git@github.com:kalbasit/swm.git` is run
- **THEN** the repository is cloned to `$CODE_ROOT/repositories/github.com/kalbasit/swm/` and the command exits 0

#### Scenario: Already cloned
- **WHEN** `swm clone git@github.com:kalbasit/swm.git` is run and the canonical path already exists with a `.git` directory
- **THEN** the command prints "already cloned at <path>" and exits 0 without calling `vcs.Clone`

#### Scenario: Clone failure
- **WHEN** `vcs.Clone` returns an error (e.g., network error, auth failure)
- **THEN** the command exits non-zero with the error message from the VCS plugin

### Requirement: swm workspace open
`swm workspace open [--story <name>]` SHALL open (or switch to) the tmux workspace for a story. The flow: (1) resolve story from `--story` flag, `$SWM_STORY` env var, or `_default`; (2) load the story's attached projects from the story store; (3) call `session.OpenWorkspace({story_name, worktree_paths: [derived paths for each project]})`; (4) if the workspace was already open, call `session.SwitchTo` for the first pane group.

#### Scenario: Open new workspace
- **WHEN** `swm workspace open --story feat-x` is run and the workspace is not currently open
- **THEN** `session.OpenWorkspace` is called with the story name and all worktree paths, and the terminal is attached to the workspace

#### Scenario: Story from environment
- **WHEN** `swm workspace open` is run with `$SWM_STORY=feat-x` set
- **THEN** the workspace for `feat-x` is opened (same as `--story feat-x`)

#### Scenario: Default story
- **WHEN** `swm workspace open` is run with no `--story` flag and no `$SWM_STORY`
- **THEN** the workspace for the `_default` story is opened

#### Scenario: Story not found
- **WHEN** `swm workspace open --story nonexistent` is run
- **THEN** the command exits non-zero with an error indicating the story was not found

#### Scenario: Story with no projects
- **WHEN** `swm workspace open --story feat-x` is run and `feat-x` has no attached projects
- **THEN** `session.OpenWorkspace` is called with an empty `worktree_paths` list; the session plugin opens an empty workspace
