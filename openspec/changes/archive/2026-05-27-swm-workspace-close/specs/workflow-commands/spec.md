## ADDED Requirements

### Requirement: swm workspace close

`swm workspace close [<name>]` SHALL close the active multiplexer workspace for a
story without removing the story record or its worktrees. The `<name>` argument is
optional. When omitted, the story name SHALL be resolved from the `$SWM_STORY`
environment variable. If `<name>` is omitted and `$SWM_STORY` is unset or empty, the
command SHALL exit non-zero with a descriptive error before performing any work. When
`<name>` is supplied it always takes precedence over `$SWM_STORY`.

The close sequence SHALL be:
1. Call `session.ListWorkspaces` to find the workspace whose `story_name` matches.
2. If no matching workspace is found, succeed with no output (idempotent).
3. Call `session.CloseWorkspace` with the matched `workspace_id`.
4. Print `closed workspace for story "<name>"` on success.

Errors from `session.ListWorkspaces` or `session.CloseWorkspace` SHALL be returned as
a non-zero exit. Unlike `swm story remove`, no best-effort cleanup is attempted — the
command either fully succeeds or reports the error.

The command SHALL NOT: remove the story JSON, remove worktrees, or run any hooks.
When no session plugin is configured, the command SHALL exit non-zero with a
descriptive error.

Shell completion for `<name>` SHALL list all story names from the story store.

#### Scenario: Close running workspace by explicit name

- **WHEN** `swm workspace close feat-x` is run and a workspace for `feat-x` is active
- **THEN** `session.ListWorkspaces` is called, the matching workspace_id is found, `session.CloseWorkspace` is called with that id, and the command prints `closed workspace for story "feat-x"` and exits zero

#### Scenario: No active workspace — succeeds silently

- **WHEN** `swm workspace close feat-x` is run and no workspace for `feat-x` exists
- **THEN** `session.ListWorkspaces` returns results but none match `feat-x`, the command exits zero with no output

#### Scenario: No arg with SWM_STORY set — closes current story workspace

- **WHEN** `swm workspace close` is run with no argument and `$SWM_STORY=feat-x`
- **THEN** the command behaves identically to `swm workspace close feat-x`

#### Scenario: No arg and SWM_STORY unset — exits with error

- **WHEN** `swm workspace close` is run with no argument and `$SWM_STORY` is unset
- **THEN** the command exits non-zero with an error indicating a story name is required

#### Scenario: Explicit arg overrides SWM_STORY

- **WHEN** `swm workspace close other-story` is run with `$SWM_STORY=feat-x`
- **THEN** the workspace for `other-story` is closed, not `feat-x`

#### Scenario: Session plugin absent — exits with error

- **WHEN** `swm workspace close feat-x` is run and no session plugin is configured
- **THEN** the command exits non-zero with a descriptive error

#### Scenario: ListWorkspaces error — exits with error

- **WHEN** `swm workspace close feat-x` is run and `session.ListWorkspaces` returns an error
- **THEN** the command exits non-zero and surfaces the error
