## MODIFIED Requirements

### Requirement: swm workspace open
`swm workspace open [<story-name>]` SHALL open (or switch to) the tmux workspace for a story. The flow depends on whether a picker plugin is configured:

Before opening the workspace the command SHALL run `hookexec.Run` for event `pre-workspace-open` with the story name set. If any `pre-workspace-open` hook returns non-zero the command SHALL abort. After the workspace is open the command SHALL run `hookexec.Run` for event `post-workspace-open`; failures are logged but do not affect the exit code.

**With picker configured:**
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
10. Call `session.SwitchTo`; if the response contains a non-empty `exec_argv`, call `syscall.Exec(exec_argv[0], exec_argv, os.Environ())` to replace the host process and hand off the terminal. The host process does not return after exec.

**Without picker configured (fallback):**
1. Run `pre-workspace-open` hooks; abort if any fail.
2. Resolve story.
3. Load all attached projects from the story store.
4. Call `session.OpenWorkspace({story_name, worktree_paths: {project_key: derived_path}})`.
5. Run `post-workspace-open` hooks.
6. If the workspace was already open, call `session.SwitchTo` for the first pane group; if the response contains a non-empty `exec_argv`, exec it as above.

#### Scenario: Interactive selection with picker — project already attached
- **WHEN** `swm workspace open feat-x` is run and picker is configured and `feat-x` has `proj-a` attached
- **THEN** `pre-workspace-open` runs, picker receives all candidates, user selects `proj-a`, `OpenWorkspace` and `OpenPaneGroup` are called, `post-workspace-open` runs

#### Scenario: Interactive selection with picker — project not yet attached
- **WHEN** `swm workspace open feat-x` is run and picker is configured and user selects `proj-b` (not yet attached)
- **THEN** `pre-worktree-create` hooks run, `vcs.CreateWorktree` is called for `proj-b`, `proj-b` is attached to the story in the store, `post-worktree-create` hooks run, then workspace and pane group are opened

#### Scenario: pre-worktree-create hook aborts worktree creation
- **WHEN** `swm workspace open feat-x` is run and user selects an unattached project and a `pre-worktree-create` hook exits non-zero
- **THEN** `vcs.CreateWorktree` is NOT called, the project is NOT attached to the story, and the command exits non-zero

#### Scenario: post-worktree-create hook fails — logged, open continues
- **WHEN** `swm workspace open feat-x` is run, a new worktree is created successfully, and a `post-worktree-create` hook exits non-zero
- **THEN** the failure is logged, the workspace open proceeds, and the command exits 0

#### Scenario: Picker cancelled by user
- **WHEN** `swm workspace open feat-x` is run and picker is configured and the user cancels selection
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

#### Scenario: Default story
- **WHEN** `swm workspace open` is run with no positional argument and no `$SWM_STORY`
- **THEN** the workspace for the `_default` story is opened

#### Scenario: Story not found
- **WHEN** `swm workspace open nonexistent` is run and no story named `nonexistent` exists
- **THEN** the command exits with a non-zero code indicating the story was not found

#### Scenario: Story with no projects and no picker
- **WHEN** `swm workspace open feat-x` is run, no picker is configured, and `feat-x` has no attached projects
- **THEN** `OpenWorkspace` is called with an empty worktree_paths map

#### Scenario: SwitchTo returns exec_argv — host execs tmux
- **WHEN** `swm workspace open --story feat-x` is run, the user is not already inside a tmux session, and `session.SwitchTo` returns a non-empty `exec_argv`
- **THEN** the host calls `syscall.Exec` with the returned argv, replacing itself with the tmux process and attaching the user to the workspace

#### Scenario: SwitchTo returns empty exec_argv — already inside tmux
- **WHEN** `swm workspace open --story feat-x` is run from inside an existing tmux session and `session.SwitchTo` returns empty `exec_argv`
- **THEN** the host does NOT call `syscall.Exec`; the tmux session switches in-place

#### Scenario: pre-workspace-open hook aborts open
- **WHEN** `swm workspace open feat-x` is run and a `pre-workspace-open` hook exits non-zero
- **THEN** the command aborts before opening the workspace and returns a non-zero exit code
