## ADDED Requirements

### Requirement: swm story attach

`swm story attach [<story-name>]` SHALL ensure the project identified by the
current working directory is attached to the resolved story, creating its
worktree and running worktree hooks only when the project is not already
attached. The command is idempotent and SHALL be safe to invoke blindly. It
SHALL NOT perform any session/multiplexer work: no session plugin is loaded and
no workspace, pane group, or `SwitchTo` is invoked.

**Story resolution** follows this precedence:

1. Positional `<story-name>` argument, if provided.
2. `$SWM_STORY` environment variable, if set and non-empty.

If neither resolves a story, the command SHALL exit non-zero with a descriptive
error before performing any work. There is no picker fallback. If the resolved
story does not exist in the story store, the command SHALL exit non-zero with a
"story not found" error (no interactive creation prompt).

**Project resolution:** the command SHALL call `vcs.DetectProjectAtPath` with the
current working directory to obtain the `ProjectID`. If the working directory is
not inside a repository the VCS plugin recognizes (e.g. no `origin` remote), the
command SHALL exit non-zero with a descriptive error. From the `ProjectID` the
host SHALL compose `RepoPath` (canonical path) and `WorktreePath` (story path)
via the layout resolver.

**Attachment logic**, after resolving the `ProjectID` and loading the story:

1. If the project is already attached to the story (present in the story's
   `projects` list), the command SHALL make no changes and exit 0. No hooks run
   and no VCS call is made.
2. If the project is not attached and no worktree exists at `WorktreePath`, the
   command SHALL, in order:
   a. Run `pre-worktree-create` hooks with full project context (`ProjectHost`,
      `ProjectPath`, `WorktreePath`, `RepoPath`); abort non-zero if any fail.
   b. If the resolved story is NOT the `default_story`, call `vcs.CreateWorktree`
      for the project using the story's `branch_name`. If it IS the
      `default_story`, skip this step (the canonical `repositories/` path is the
      checkout).
   c. Attach the project to the story in the store.
   d. Run `post-worktree-create` hooks with the same project context; failures
      are logged but do not affect the exit code.
3. If the project is not attached but a worktree already exists at `WorktreePath`
   (bookkeeping drift), the command SHALL reconcile by attaching the project to
   the store only: `vcs.CreateWorktree` SHALL NOT be called and the
   `pre`/`post-worktree-create` hooks SHALL NOT run. The command exits 0.

If `store.Update` reports the project is already attached (a concurrent attach
won the race), the command SHALL treat that as success and exit 0.

#### Scenario: Attaches an unattached project and creates its worktree
- **WHEN** `swm story attach feat-x` is run from inside a repository whose project is not attached to `feat-x`, `feat-x` exists, and no worktree exists for it
- **THEN** `vcs.DetectProjectAtPath` resolves the project, `pre-worktree-create` hooks run, `vcs.CreateWorktree` is called with the story's `branch_name`, the project is attached in the store, `post-worktree-create` hooks run, and the command exits 0

#### Scenario: Already attached — no-op
- **WHEN** `swm story attach feat-x` is run and the current project is already in `feat-x`'s `projects` list
- **THEN** no hooks run, `vcs.CreateWorktree` is NOT called, the store is unchanged, and the command exits 0

#### Scenario: Worktree exists but project not attached — reconcile only
- **WHEN** `swm story attach feat-x` is run, the current project is not in `feat-x`'s `projects` list, but a worktree already exists at the project's story worktree path
- **THEN** `vcs.CreateWorktree` is NOT called, no `pre`/`post-worktree-create` hooks run, the project is attached in the store, and the command exits 0

#### Scenario: Story resolved from $SWM_STORY
- **WHEN** `swm story attach` is run with no positional argument and `$SWM_STORY=feat-x` set
- **THEN** the command behaves identically to `swm story attach feat-x`

#### Scenario: Positional argument overrides $SWM_STORY
- **WHEN** `swm story attach other-story` is run with `$SWM_STORY=feat-x` set
- **THEN** the project is attached to `other-story`, not `feat-x`

#### Scenario: No arg and $SWM_STORY unset — exits with error
- **WHEN** `swm story attach` is run with no positional argument and `$SWM_STORY` unset or empty
- **THEN** the command exits non-zero with an error indicating a story name is required, and no hooks, VCS calls, or store writes occur

#### Scenario: Story not found — exits with error
- **WHEN** `swm story attach nonexistent` is run and no story named `nonexistent` exists
- **THEN** the command exits non-zero with a "story not found" error and no worktree is created

#### Scenario: Not inside a repository — exits with error
- **WHEN** `swm story attach feat-x` is run from a directory that `vcs.DetectProjectAtPath` does not recognize as a repository
- **THEN** the command exits non-zero with a descriptive error and makes no store writes

#### Scenario: Resolves project from a subdirectory of the repo
- **WHEN** `swm story attach feat-x` is run from a nested subdirectory of the repository (not its root)
- **THEN** `vcs.DetectProjectAtPath` still resolves the correct `ProjectID` and the project is attached to `feat-x`

#### Scenario: Default story — worktree creation skipped
- **WHEN** `swm story attach` resolves the `default_story` and the current project is not yet attached
- **THEN** `pre-worktree-create` hooks run, `vcs.CreateWorktree` is NOT called, the project is attached in the store, `post-worktree-create` hooks run, and the command exits 0

#### Scenario: pre-worktree-create hook aborts attach
- **WHEN** `swm story attach feat-x` is run for an unattached project with no existing worktree and a `pre-worktree-create` hook exits non-zero
- **THEN** `vcs.CreateWorktree` is NOT called, the project is NOT attached, and the command exits non-zero

#### Scenario: post-worktree-create hook fails — logged, command succeeds
- **WHEN** `swm story attach feat-x` creates a worktree successfully and a `post-worktree-create` hook exits non-zero
- **THEN** the failure is logged, the project remains attached, and the command exits 0

#### Scenario: No session work is performed
- **WHEN** `swm story attach feat-x` is run to any effect (create, reconcile, or no-op)
- **THEN** no session plugin is loaded and `OpenWorkspace`, `OpenPaneGroup`, and `SwitchTo` are never called

#### Scenario: Shell completion lists story names
- **WHEN** shell completion is requested for the `<story-name>` argument
- **THEN** all story names from the story store are offered
