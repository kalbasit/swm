## MODIFIED Requirements

### Requirement: swm story remove
`swm story remove [<name>] [--force]` SHALL remove a story and all its worktrees. The `<name>` argument is optional. When omitted, the story name SHALL be resolved from the `$SWM_STORY` environment variable. If `<name>` is omitted and `$SWM_STORY` is unset or empty, the command SHALL exit non-zero with a descriptive error before performing any work. When `<name>` is supplied it always takes precedence over `$SWM_STORY`.

Without `--force`, a confirmation prompt MUST be shown listing all attached projects. The removal sequence SHALL be:
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

#### Scenario: No arg with SWM_STORY set — removes current story
- **WHEN** `swm story remove` is run with no positional argument and `$SWM_STORY=feat-x` is set in the environment
- **THEN** the story named `feat-x` is removed (same behaviour as `swm story remove feat-x`)

#### Scenario: No arg with SWM_STORY set and --force — skips prompt
- **WHEN** `swm story remove --force` is run with `$SWM_STORY=feat-x` set
- **THEN** the story `feat-x` is removed without a confirmation prompt

#### Scenario: No arg and SWM_STORY unset — exits with error
- **WHEN** `swm story remove` is run with no positional argument and `$SWM_STORY` is unset or empty
- **THEN** the command exits non-zero with an error message indicating that a story name is required, and no removal is attempted

#### Scenario: Explicit arg overrides SWM_STORY
- **WHEN** `swm story remove other-story` is run with `$SWM_STORY=feat-x` set
- **THEN** the story `other-story` is removed, not `feat-x`
