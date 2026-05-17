## MODIFIED Requirements

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
2. Resolve story (see precedence above).
3. Build a project candidate list: all projects attached to the resolved story plus all
   repositories discovered under `$CODE_ROOT/repositories/`.
4. Stream project candidates to `picker.Pick`; each candidate's `display` and `key` are its
   project ID string (e.g. `github.com/kalbasit/swm`).
5. Receive the selected project ID.
6. If the selected project is NOT already attached to the story: call `vcs.CreateWorktree` and
   attach to the story store.
7. Call `session.OpenWorkspace`.
8. Call `session.OpenPaneGroup` with the worktree path for the selected project.
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
- **THEN** `vcs.CreateWorktree` is called for `proj-b`, `proj-b` is attached to the story in the store, then workspace and pane group are opened

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

## ADDED Requirements

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
