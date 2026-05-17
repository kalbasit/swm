## MODIFIED Requirements

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
10. Call `session.SwitchTo`; if the response contains a non-empty `exec_argv`, call `syscall.Exec(exec_argv[0], exec_argv, os.Environ())` to replace the host process and hand off the terminal. The host process does not return after exec.

**Without picker configured (fallback):**
1. Run `pre-workspace-open` hooks; abort if any fail.
2. Resolve story.
3. Load all attached projects from the story store.
4. Call `session.OpenWorkspace({story_name, worktree_paths: {project_key: derived_path}})`.
5. If the workspace was already open, call `session.SwitchTo` for the first pane group; if the response contains a non-empty `exec_argv`, exec it as above.
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

#### Scenario: SwitchTo returns exec_argv — host execs tmux
- **WHEN** `swm workspace open --story feat-x` is run, the user is not already inside a tmux session, and `session.SwitchTo` returns a non-empty `exec_argv`
- **THEN** the host calls `syscall.Exec` with the returned argv, replacing itself with the tmux process and attaching the user to the workspace

#### Scenario: SwitchTo returns empty exec_argv — already inside tmux
- **WHEN** `swm workspace open --story feat-x` is run from inside an existing tmux session and `session.SwitchTo` returns empty `exec_argv`
- **THEN** the host does NOT call `syscall.Exec`; the tmux session switches in-place
