## MODIFIED Requirements

### Requirement: swm workspace open
`swm workspace open [--story <name>]` SHALL open (or switch to) the tmux workspace for a story. The flow depends on whether a picker plugin is configured:

**With picker configured:**
1. Resolve story from `--story` flag, `$SWM_STORY` env var, or `_default`.
2. Build a candidate list: all projects already attached to the story plus all repositories discovered under `$CODE_ROOT/repositories/` via `host.ListProjects`.
3. Stream candidates to `picker.Pick`; each candidate's `display` is its project ID string (e.g. `github.com/kalbasit/swm`) and `id` is the same string.
4. Receive the selected project ID from the picker.
5. If the selected project is NOT already attached to the story: call `vcs.CreateWorktree` for that project and attach it to the story in the story store.
6. Call `session.OpenPaneGroup` with the derived worktree path for the selected project.

**Without picker configured (fallback):**
1. Resolve story.
2. Load all attached projects from the story store.
3. Call `session.OpenWorkspace({story_name, worktree_paths: [derived paths for each project]})`.
4. If the workspace was already open, call `session.SwitchTo` for the first pane group.

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
- **THEN** `session.OpenWorkspace` is called with an empty `worktree_paths` list; the session plugin opens an empty workspace
