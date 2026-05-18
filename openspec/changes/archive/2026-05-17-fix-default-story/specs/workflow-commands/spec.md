---
capability: workflow-commands
change: fix-default-story
---

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
7. Call `session.OpenWorkspace({story_name, worktree_paths: {selected_key: worktree_path}})`.
8. Call `session.OpenPaneGroup({workspace_id, project_id, worktree_path})`.
9. Build a `SwitchToRequest` for the resulting pane group.
10. If `--kill-pane` is set and a multiplexer origin (pane ID + workspace ID) is detected:
    a. Set `close_origin_pane_id` and `close_origin_workspace_id` in the `SwitchToRequest`.
11. Call `session.SwitchTo`; if `exec_argv` is non-empty, exec into that command.
12. Run `post-workspace-open` hooks; failures are logged but do not abort.

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
- **THEN** the story picker is presented with all stories

#### Scenario: Story picker entries include _default as last entry
- **WHEN** the story picker is shown and both `_default` and feature stories exist
- **THEN** `_default` appears as the last entry with display text starting with `_default (main repo)`, and all feature stories appear before it sorted by `CreatedAt` descending

#### Scenario: Story picker entry omits branch when equal to story name
- **WHEN** the story picker is shown and a story's branch name equals its story name
- **THEN** the display text does not include the branch name in parentheses

#### Scenario: Story picker entry shows branch when it differs from story name
- **WHEN** the story picker is shown and a story's branch name differs from the story name
- **THEN** the display text includes the branch name in parentheses after the story name

#### Scenario: Story picker entry truncates projects to fit terminal width
- **WHEN** the story picker is shown and the full project list would exceed terminal width
- **THEN** the project list is trimmed with ` …` appended

#### Scenario: Story picker cancelled by user
- **WHEN** the story picker is shown and the user cancels (EOF / no selection)
- **THEN** the command exits zero with no workspace opened

#### Scenario: Story picker unavailable — falls back to default story
- **WHEN** `swm workspace open` is run with no arg, no env, and no picker is configured
- **THEN** the `_default` story is opened using the no-picker fallback path

#### Scenario: Story picker returns FailedPrecondition — falls back to default story
- **WHEN** `swm workspace open` is run with no arg, no env, picker is configured but no TTY is available
- **THEN** the story picker returns `FailedPrecondition`, and the command opens `_default` using the no-picker fallback path

#### Scenario: Arg provided — story picker skipped
- **WHEN** `swm workspace open feat-x` is run
- **THEN** the story picker is not invoked and `feat-x` is opened directly

#### Scenario: $SWM_STORY set — story picker skipped
- **WHEN** `swm workspace open` is run with `$SWM_STORY=feat-x` and no positional argument
- **THEN** the story picker is not invoked and `feat-x` is opened directly

#### Scenario: Interactive selection with picker — project already attached
- **WHEN** `swm workspace open feat-x` is run and picker is configured and `feat-x` has `proj-a` attached
- **THEN** `vcs.CreateWorktree` is NOT called and the workspace is opened directly for `proj-a`

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
- **THEN** the error is logged and the workspace open continues normally

#### Scenario: Project picker cancelled by user
- **WHEN** `swm workspace open feat-x` is run and picker is configured and user cancels the project picker
- **THEN** the command exits zero with no workspace opened

#### Scenario: No picker configured — opens all attached projects
- **WHEN** `swm workspace open feat-x` is run and no picker is configured
- **THEN** `OpenWorkspace` is called with all attached projects' worktree paths

#### Scenario: Story from environment
- **WHEN** `swm workspace open` is run with `$SWM_STORY=feat-x` set and no picker
- **THEN** `feat-x` is opened directly without showing the story picker

#### Scenario: Positional argument overrides environment variable
- **WHEN** `swm workspace open feat-y` is run with `$SWM_STORY=feat-x` set
- **THEN** `feat-y` is opened (positional arg takes precedence)

#### Scenario: Story not found
- **WHEN** `swm workspace open nonexistent` is run
- **THEN** the command exits non-zero with a "story not found" error

#### Scenario: Story with no projects and no picker
- **WHEN** `swm workspace open feat-x` is run, no picker is configured, and `feat-x` has no attached projects
- **THEN** `OpenWorkspace` is called with an empty worktree_paths map

#### Scenario: SwitchTo returns exec_argv — host execs
- **WHEN** `session.SwitchTo` returns a non-empty `exec_argv`
- **THEN** the process execs into that command (replaces itself)

#### Scenario: SwitchTo returns empty exec_argv — already inside session
- **WHEN** `session.SwitchTo` returns an empty `exec_argv`
- **THEN** the command exits zero without exec-ing

#### Scenario: pre-workspace-open hook aborts open
- **WHEN** a `pre-workspace-open` hook exits non-zero before any workspace action
- **THEN** the command exits non-zero without opening any workspace

#### Scenario: --kill-pane closes origin pane after switch
- **WHEN** `swm workspace open --kill-pane` is run inside an existing tmux session with a detected pane ID
- **THEN** `SwitchToRequest` includes `close_origin_pane_id` and `close_origin_workspace_id`

#### Scenario: --kill-pane is no-op when TMUX_PANE is unset
- **WHEN** `swm workspace open --kill-pane` is run and `TMUX_PANE` is not set
- **THEN** `SwitchToRequest` does NOT include `close_origin_pane_id`

#### Scenario: --kill-pane is no-op when CurrentContext fails
- **WHEN** `swm workspace open --kill-pane` is run and `session.CurrentContext` returns an error
- **THEN** `SwitchToRequest` does NOT include `close_origin_pane_id`

#### Scenario: --kill-pane with exec path
- **WHEN** `swm workspace open --kill-pane` is run and `session.SwitchTo` returns a non-empty `exec_argv`
- **THEN** `SwitchToRequest` includes `close_origin_pane_id` and the process execs into `exec_argv`
