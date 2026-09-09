## ADDED Requirements

### Requirement: swm story show

`swm story show [<story-name>] [--json]` SHALL report one story: its name, its branch name, when it was created, and every attached project with the absolute worktree path that project resolves to on this host.

Story resolution SHALL follow the positional argument, then `$SWM_STORY`, then the configured `default_story`. A story that does not exist SHALL exit non-zero naming it.

Worktree paths SHALL be resolved through the same layout resolver the rest of the CLI uses, so that what `show` reports and what `workspace ensure` opens cannot disagree. The command SHALL NOT consult a session plugin or a multiplexer: it reports what swm records, not what happens to be running.

A project SHALL be identified by the same `<host>/<segments...>` key used everywhere a project is named outside a multiplexer, so a caller can match what this reports against what it asked for.

#### Scenario: Reports a story's branch name and worktree paths

- **WHEN** `swm story show feat-x --json` is run for a story with two attached projects
- **THEN** the output carries the story's branch name and both projects, each with its project key and the absolute worktree path for that story on this host

#### Scenario: The default story resolves to the canonical clone

- **WHEN** `swm story show` is run for the configured default story
- **THEN** each project's reported path is its canonical clone, because the default story has no separate worktree

#### Scenario: A story with no attached projects

- **WHEN** `swm story show feat-x --json` is run for a story nothing is attached to
- **THEN** the story is reported with its branch name and an empty project list, rather than an error

#### Scenario: An unknown story is refused

- **WHEN** `swm story show no-such-story` is run
- **THEN** the command exits non-zero naming the story and reports no record

#### Scenario: Story resolution falls through to the default

- **WHEN** `swm story show` is run with no argument and `$SWM_STORY` unset
- **THEN** the configured `default_story` is reported

#### Scenario: No multiplexer is consulted

- **WHEN** `swm story show feat-x` is run while no workspace is open
- **THEN** the story is reported in full, because the paths come from swm's records rather than from a running session

#### Scenario: Human-readable output without --json

- **WHEN** `swm story show feat-x` is run without `--json`
- **THEN** the story's name, branch name and each project with its path are printed in a form a person can read
