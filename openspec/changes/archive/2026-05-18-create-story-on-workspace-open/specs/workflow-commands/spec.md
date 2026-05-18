## MODIFIED Requirements

### Requirement: swm workspace open
`swm workspace open [<story-name>] [--kill-pane]` SHALL open (or switch to) the workspace for a story. Story
resolution follows this precedence:

1. Positional `<story-name>` argument, if provided.
2. `$SWM_STORY` environment variable, if set and non-empty.
3. If neither (1) nor (2) resolves a story AND a picker plugin is configured AND a TTY is
   available: present a **story picker** listing all stories so the user can select one
   interactively.
4. If the story picker is unavailable (no picker plugin, no TTY, or picker returns
   `FailedPrecondition`): fall back to the `default_story` from config (`_default`).

**Story not found — interactive creation:**
When a story name resolved via step (1) or (2) does not exist in the story store:
- If stdin is a TTY: prompt the user `Story '<name>' does not exist. Create it? [y/N]: `
  (written to stderr).
  - If the user answers `y` or `Y`: create the story (running `pre-story-create` and
    `post-story-create` hooks as `swm story create` would), then continue with the open flow.
  - Any other answer: exit non-zero with a "story not found" error.
- If stdin is NOT a TTY: exit non-zero with a "story not found" error (unchanged behavior).

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
   b. **If `storyName` is NOT the `default_story`**: call `vcs.CreateWorktree` for that project. If `storyName` IS the `default_story`, skip this step — the canonical `repositories/` path already exists as the main git checkout.
   c. Attach the project to the story in the story store.
   d. Run `post-worktree-create` hooks with the same project context; failures are logged but do not abort.
7. Call `session.OpenWorkspace` to ensure the workspace is active.
8. Call `session.OpenPaneGroup` with the derived worktree path for the selected project.
9. Run `post-workspace-open` hooks.
10. Build `SwitchToRequest`:
    - When `--kill-pane` is set AND `$TMUX_PANE` is non-empty: call `session.CurrentContext()` to get the current `workspace_id`; set `close_origin_workspace_id` and `close_origin_pane_id` on the request. If `CurrentContext()` fails or returns an empty `workspace_id`, omit the origin fields (silent no-op).
    - Otherwise: omit `close_origin_workspace_id` and `close_origin_pane_id`.
11. Call `session.SwitchTo`; if the response contains a non-empty `exec_argv`, call `syscall.Exec` to replace the host process.

**Without picker configured (fallback):**
1. Run `pre-workspace-open` hooks; abort if any fail.
2. Resolve story (arg → env → default).
3. Load all attached projects.
4. Call `session.OpenWorkspace({story_name, worktree_paths: {project_key: derived_path}})`.
5. Run `post-workspace-open` hooks.
6. Build `SwitchToRequest` applying the same `--kill-pane` logic as step 10 above.
7. Call `session.SwitchTo` for the first pane group; exec if `exec_argv` is non-empty.

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

#### Scenario: Interactive selection with picker — _default story, project not yet attached
- **WHEN** `swm workspace open` is run, `_default` is the resolved story, picker is configured, and the user selects `proj-b` which is not yet attached to `_default`
- **THEN** `pre-worktree-create` hooks run, `vcs.CreateWorktree` is NOT called (canonical path already exists), `proj-b` is attached to the story in the store, `post-worktree-create` hooks run, then workspace and pane group are opened

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

#### Scenario: Story not found — user confirms creation
- **WHEN** `swm workspace open nonexistent` is run, no story named `nonexistent` exists, stdin is a TTY, and the user answers `y`
- **THEN** `pre-story-create` hooks run, the story is created in the store, `post-story-create` hooks run, and the open flow continues for the newly created story

#### Scenario: Story not found — user declines creation
- **WHEN** `swm workspace open nonexistent` is run, no story named `nonexistent` exists, stdin is a TTY, and the user answers anything other than `y`/`Y`
- **THEN** the command exits with a non-zero code indicating the story was not found

#### Scenario: Story not found — non-TTY stdin
- **WHEN** `swm workspace open nonexistent` is run, no story named `nonexistent` exists, and stdin is NOT a TTY
- **THEN** the command exits immediately with a non-zero code indicating the story was not found (no prompt shown)

#### Scenario: Story not found — pre-story-create hook aborts creation
- **WHEN** `swm workspace open nonexistent` is run, stdin is a TTY, the user confirms creation, and a `pre-story-create` hook exits non-zero
- **THEN** the story is NOT created and the command exits non-zero

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

#### Scenario: --kill-pane closes origin pane after switch
- **WHEN** `swm workspace open feat-x --kill-pane` is run from inside a tmux session with `$TMUX_PANE=%5` set
- **THEN** `CurrentContext()` is called to get the origin `workspace_id`, `SwitchToRequest` is built with `close_origin_workspace_id` and `close_origin_pane_id = "%5"`, and `SwitchTo` is called with those fields set

#### Scenario: --kill-pane is no-op when TMUX_PANE is unset
- **WHEN** `swm workspace open feat-x --kill-pane` is run and `$TMUX_PANE` is empty
- **THEN** `CurrentContext()` is NOT called, `close_origin_pane_id` is omitted from `SwitchToRequest`, and the switch proceeds normally without killing any pane

#### Scenario: --kill-pane is no-op when CurrentContext fails
- **WHEN** `swm workspace open feat-x --kill-pane` is run, `$TMUX_PANE` is set, but `session.CurrentContext()` returns an error
- **THEN** the origin fields are omitted from `SwitchToRequest` and the switch proceeds normally

#### Scenario: --kill-pane with exec path
- **WHEN** `swm workspace open feat-x --kill-pane` is run outside any tmux session and `SwitchTo` returns a non-empty `exec_argv`
- **THEN** `SwitchToRequest` is built with origin fields set (if `$TMUX_PANE` and `CurrentContext()` succeed), and after `syscall.Exec` the plugin has already killed the origin pane before responding
