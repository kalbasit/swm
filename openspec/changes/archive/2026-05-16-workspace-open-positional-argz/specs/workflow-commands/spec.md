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
- **WHEN** `swm workspace open feat-x` is run and picker is configured and `feat-x` has `proj-a` attached
- **THEN** `pre-workspace-open` runs, picker receives all candidates, user selects `proj-a`, `OpenWorkspace` and `OpenPaneGroup` are called, `post-workspace-open` runs

#### Scenario: Interactive selection with picker — project not yet attached
- **WHEN** `swm workspace open feat-x` is run and picker is configured and user selects `proj-b` (not yet attached)
- **THEN** `vcs.CreateWorktree` is called for `proj-b`, `proj-b` is attached to the story in the store, then workspace and pane group are opened

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

#### Scenario: pre-workspace-open hook aborts open
- **WHEN** `swm workspace open feat-x` is run and a `pre-workspace-open` hook exits non-zero
- **THEN** the command aborts before opening the workspace and returns a non-zero exit code
