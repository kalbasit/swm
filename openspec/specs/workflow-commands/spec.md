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
`swm workspace open [<story-name>]` SHALL open (or switch to) the workspace for a story. Story
resolution follows this precedence:

1. Positional `<story-name>` argument, if provided.
2. `$SWM_STORY` environment variable, if set and non-empty.
3. If neither (1) nor (2) resolves a story AND a picker plugin is configured AND a TTY is
   available: present a **story picker** listing all stories so the user can select one
   interactively.
4. If the story picker is unavailable (no picker plugin, no TTY, or picker returns
   `FailedPrecondition`): fall back to the `default_story` from config (`_default`).

Before opening the workspace the command SHALL run `hookexec.Run` for event
`pre-workspace-open` with the story name set. If any `pre-workspace-open` hook returns
non-zero the command SHALL abort. After the workspace is open the command SHALL run
`hookexec.Run` for event `post-workspace-open`; failures are logged but do not affect the exit
code.

**Story picker entry format:**

Each `PickItem` sent to the picker plugin SHALL have:
- `key`: the raw story name (e.g. `_default`, `feat/workspace-open-picker`)
- `display`: a terminal-width-aware formatted string:
  `<story-name>[ (<branch-name>)]   <age> ago   <project1> · <project2> · …`
  - Branch name in parentheses is shown only when it differs from the story name.
  - `_default` story MUST display as `_default (main repo)` regardless of branch name.
  - Age is formatted as a rounded-up single unit: `Xm`, `Xh`, `Xd`, `Xw`, `Xmo`, `Xy`.
  - Projects are joined with ` · `; the list is trimmed with ` …` if it exceeds available width.
  - The host detects terminal width via `/dev/tty` → `$COLUMNS` env var → 120 columns default.
  - Truncation priority (right-to-left): projects list → branch name → story name.

Stories SHALL be sent to the picker sorted by `CreatedAt` descending (most recent first), with
`_default` pinned last.

**With picker configured and story resolved (from arg, env, or picker selection):**
1. Run `pre-workspace-open` hooks; abort if any fail.
2. Resolve story from positional `<story-name>` argument, `$SWM_STORY` env var, or `_default`.
3. Build a candidate list: all projects already attached to the story plus all repositories discovered under `$CODE_ROOT/repositories/` via `host.ListProjects`.
4. Stream candidates to `picker.Pick`; each candidate's `display` is its project ID string (e.g. `github.com/kalbasit/swm`) and `key` is the same string.
5. Receive the selected project ID from the picker.
6. If the selected project is NOT already attached to the story:
   a. Run `pre-worktree-create` hooks with full project context (`ProjectHost`, `ProjectPath`, `WorktreePath`, `RepoPath`); abort if any fail.
   b. Call `vcs.CreateWorktree` for that project.
   c. Attach the project to the story in the story store.
   d. Run `post-worktree-create` hooks with the same project context; failures are logged but do not abort.
7. Call `session.OpenWorkspace` to ensure the workspace is active.
8. Call `session.OpenPaneGroup` with the derived worktree path for the selected project.
9. Run `post-workspace-open` hooks.
10. Call `session.SwitchTo`; if the response contains a non-empty `exec_argv`, call
    `syscall.Exec` to replace the host process.

**Without picker configured (fallback):**
1. Run `pre-workspace-open` hooks; abort if any fail.
2. Resolve story (arg → env → default).
3. Load all attached projects.
4. Call `session.OpenWorkspace({story_name, worktree_paths: {project_key: derived_path}})`.
5. Run `post-workspace-open` hooks.
6. Call `session.SwitchTo` for the first pane group; exec if `exec_argv` is non-empty.

#### Scenario: No arg, no env — story picker shown
- **WHEN** `swm workspace open` is run with no positional argument, `$SWM_STORY` is unset, picker is configured, and a TTY is available
- **THEN** the command streams all stories to `picker.Pick` and waits for the user to select one before proceeding to project selection

#### Scenario: Story picker entries include _default as last entry
- **WHEN** the story picker is shown and both `_default` and feature stories exist
- **THEN** `_default` appears as the last entry with display text starting with `_default (main repo)`, and all feature stories appear before it sorted by `CreatedAt` descending

#### Scenario: Story picker entry omits branch when equal to story name
- **WHEN** a story named `feat/my-feature` has `branch_name = "feat/my-feature"`
- **THEN** its picker display shows `feat/my-feature` with no parenthetical branch name

#### Scenario: Story picker entry shows branch when it differs from story name
- **WHEN** a story named `jira-42` has `branch_name = "fix/JIRA-42-crash"`
- **THEN** its picker display shows `jira-42 (fix/JIRA-42-crash)   <age>   <projects>`

#### Scenario: Story picker entry truncates projects to fit terminal width
- **WHEN** the terminal is 80 columns wide and a story has many attached projects
- **THEN** the display string ends with ` …` and does not exceed 80 characters

#### Scenario: Story picker cancelled by user
- **WHEN** the story picker is shown and the user presses Escape or Ctrl-C
- **THEN** the command exits 0 and no workspace is opened

#### Scenario: Story picker unavailable — falls back to default story
- **WHEN** `swm workspace open` is run with no arg, no env, and no picker is configured
- **THEN** the `_default` story is opened using the no-picker fallback path

#### Scenario: Story picker returns FailedPrecondition — falls back to default story
- **WHEN** `swm workspace open` is run with no arg, no env, picker is configured but no TTY is available
- **THEN** the story picker returns `FailedPrecondition`, and the command opens `_default` using the no-picker fallback path

#### Scenario: Arg provided — story picker skipped
- **WHEN** `swm workspace open feat-x` is run
- **THEN** the story picker is NOT shown; `feat-x` is used directly and the project picker runs for `feat-x`

#### Scenario: $SWM_STORY set — story picker skipped
- **WHEN** `swm workspace open` is run with `$SWM_STORY=feat-x` and no positional argument
- **THEN** the story picker is NOT shown; `feat-x` is used directly

#### Scenario: Interactive selection with picker — project already attached
- **WHEN** `swm workspace open feat-x` is run and picker is configured and `feat-x` has `proj-a` attached
- **THEN** `pre-workspace-open` runs, picker receives all project candidates, user selects `proj-a`, `OpenWorkspace` and `OpenPaneGroup` are called, `post-workspace-open` runs

#### Scenario: Interactive selection with picker — project not yet attached
- **WHEN** `swm workspace open feat-x` is run and picker is configured and user selects `proj-b` (not yet attached)
- **THEN** `pre-worktree-create` hooks run, `vcs.CreateWorktree` is called for `proj-b`, `proj-b` is attached to the story in the store, `post-worktree-create` hooks run, then workspace and pane group are opened

#### Scenario: pre-worktree-create hook aborts worktree creation
- **WHEN** `swm workspace open feat-x` is run and user selects an unattached project and a `pre-worktree-create` hook exits non-zero
- **THEN** `vcs.CreateWorktree` is NOT called, the project is NOT attached to the story, and the command exits non-zero

#### Scenario: post-worktree-create hook fails — logged, open continues
- **WHEN** `swm workspace open feat-x` is run, a new worktree is created successfully, and a `post-worktree-create` hook exits non-zero
- **THEN** the failure is logged, the workspace open proceeds, and the command exits 0

#### Scenario: Project picker cancelled by user
- **WHEN** a story is resolved and the project picker is shown but the user cancels selection
- **THEN** the command exits with code 0 and no workspace is opened

#### Scenario: No picker configured — opens all attached projects
- **WHEN** `swm workspace open feat-x` is run and no picker is configured
- **THEN** `OpenWorkspace` is called with all attached projects' worktree paths

#### Scenario: Story from environment
- **WHEN** `swm workspace open` is run with `$SWM_STORY=feat-x` set and no positional argument
- **THEN** the workspace for `feat-x` is opened

#### Scenario: Positional argument overrides environment variable
- **WHEN** `swm workspace open other-story` is run with `$SWM_STORY=feat-x` set
- **THEN** the workspace for `other-story` is opened (positional arg takes priority)

#### Scenario: Story not found
- **WHEN** `swm workspace open nonexistent` is run and no story named `nonexistent` exists
- **THEN** the command exits with a non-zero code indicating the story was not found

#### Scenario: Story with no projects and no picker
- **WHEN** `swm workspace open feat-x` is run, no picker is configured, and `feat-x` has no attached projects
- **THEN** `OpenWorkspace` is called with an empty worktree_paths map

#### Scenario: SwitchTo returns exec_argv — host execs
- **WHEN** `swm workspace open` is run outside an existing session and `session.SwitchTo` returns a non-empty `exec_argv`
- **THEN** the host calls `syscall.Exec` with the returned argv, replacing itself

#### Scenario: SwitchTo returns empty exec_argv — already inside session
- **WHEN** `swm workspace open` is run from inside an existing session and `session.SwitchTo` returns empty `exec_argv`
- **THEN** the host does NOT call `syscall.Exec`; the session switches in-place

#### Scenario: pre-workspace-open hook aborts open
- **WHEN** `swm workspace open feat-x` is run and a `pre-workspace-open` hook exits non-zero
- **THEN** the command aborts before opening the workspace and returns a non-zero exit code

### Requirement: swm workspace list
`swm workspace list` SHALL print a tree of all workspaces and their attached projects to stdout. Workspaces are listed in lexicographic order by name; projects within each workspace are listed in lexicographic order by their canonical path (`host/segments...`). The `_default` story is excluded from output. The output uses box-drawing glyphs:
```
story-1
├── github.com/a/b
└── github.com/c/d
story-2
└── github.com/e/f
```
Workspaces with no projects are printed as a plain name with no children. Exit code is 0 on success, non-zero on store error.

#### Scenario: No workspaces
- **WHEN** `swm workspace list` is run and the story store contains no stories
- **THEN** the command exits zero and prints nothing to stdout

#### Scenario: Workspace with no projects
- **WHEN** `swm workspace list` is run and story `feat-x` exists with no attached projects
- **THEN** the command exits zero and prints `feat-x` as a top-level entry with no project children

#### Scenario: Single workspace with one project
- **WHEN** `swm workspace list` is run and story `feat-x` has one attached project `github.com/a/b`
- **THEN** the command exits zero and prints a tree with `feat-x` as the root and `github.com/a/b` beneath it using `└──`

#### Scenario: Multiple workspaces with multiple projects
- **WHEN** `swm workspace list` is run and stories `alpha` and `beta` exist, with `alpha` having projects `github.com/a/b` and `github.com/c/d`, and `beta` having project `github.com/e/f`
- **THEN** the output lists `alpha` before `beta` (lexicographic), `github.com/a/b` before `github.com/c/d` within `alpha` (with `├──`), and the last project in each workspace uses `└──`

#### Scenario: Store error
- **WHEN** `swm workspace list` is run and the story store returns an error
- **THEN** the command exits non-zero and prints the error to stderr

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

### Requirement: Plugin stderr forwarding
The host plugin manager SHALL forward each plugin process's stderr to the host's own
stderr so that plugin panics, `os.Exit` messages, and runtime errors are visible to the
operator without requiring a separate debug session.

Forwarding SHALL be enabled for every plugin capability (session, vcs, picker, forge)
and SHALL be set up at plugin launch time, before the first gRPC call is made.

#### Scenario: Plugin writes to stderr before crashing
- **WHEN** a plugin binary writes a message to its stderr and then exits non-zero
- **THEN** the message appears on the host's stderr, prefixed with the plugin binary path

#### Scenario: Plugin stderr forwarded for all capabilities
- **WHEN** swm launches a session, vcs, picker, or forge plugin
- **THEN** any output the plugin writes to stderr is forwarded to the host's stderr stream

#### Scenario: Healthy plugin produces no extra output
- **WHEN** a plugin runs successfully and writes nothing to stderr
- **THEN** the host's stderr receives no additional output from the plugin

### Requirement: Age formatting (rounded-up)
The host SHALL format a story's age as a single rounded-up unit with an `ago` suffix.
Thresholds (round up to next unit when any remainder exists):
- < 1 hour → `Xm ago` (minutes, minimum 1m)
- < 1 day → `Xh ago` (hours)
- < 1 week → `Xd ago` (days)
- < 1 month (4 weeks) → `Xw ago` (weeks)
- < 1 year → `Xmo ago` (months)
- ≥ 1 year → `Xy ago` (years)

#### Scenario: Sub-hour age rounds up to minutes
- **WHEN** a story was created 47 minutes and 30 seconds ago
- **THEN** the formatted age is `48m ago`

#### Scenario: Sub-day age rounds up to hours
- **WHEN** a story was created 23 hours and 1 minute ago
- **THEN** the formatted age is `24h ago`

#### Scenario: Sub-week age rounds up to days
- **WHEN** a story was created 6 days and 2 hours ago
- **THEN** the formatted age is `7d ago`

#### Scenario: Sub-month age rounds up to weeks
- **WHEN** a story was created 13 days ago (exactly)
- **THEN** the formatted age is `2w ago`

#### Scenario: Exactly one year
- **WHEN** a story was created exactly 365 days ago
- **THEN** the formatted age is `1y ago`

### Requirement: Terminal width detection
The host SHALL detect the terminal width using the following fallback chain:
1. `term.GetSize` on `/dev/tty` — if the file can be opened and returns width > 0.
2. `$COLUMNS` environment variable parsed as a positive integer.
3. Default of 120 columns.

#### Scenario: /dev/tty provides width
- **WHEN** stdout is piped (e.g., into fzf) but `/dev/tty` is available and reports 132 columns
- **THEN** the host uses 132 as the terminal width

#### Scenario: /dev/tty unavailable, $COLUMNS set
- **WHEN** `/dev/tty` cannot be opened and `$COLUMNS=80` is set
- **THEN** the host uses 80 as the terminal width

#### Scenario: Both unavailable — default used
- **WHEN** `/dev/tty` cannot be opened and `$COLUMNS` is unset
- **THEN** the host uses 120 as the terminal width
