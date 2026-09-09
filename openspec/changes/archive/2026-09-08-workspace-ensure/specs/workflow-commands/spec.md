## ADDED Requirements

### Requirement: swm workspace ensure

`swm workspace ensure [<story-name>] [--json]` SHALL make a story's workspace and pane groups exist and then exit, without attaching to them.

Story resolution SHALL follow the positional `<story-name>` argument, then `$SWM_STORY`, then the configured `default_story`. It SHALL NOT present a story picker and SHALL NOT prompt on stdin, whether or not a terminal is attached: a command whose behaviour depends on how it was invoked cannot be relied on by a caller that has no terminal to offer.

A story that does not exist SHALL exit non-zero naming it, and SHALL NOT be created. Creating a story is `swm story create`, and a caller that wants one can say so.

The command SHALL call `session.OpenWorkspace` for the story with the derived worktree path of every attached project, then `session.OpenPaneGroup` for each of those projects. It SHALL NOT call `session.SwitchTo` and SHALL NOT replace its own process.

The command SHALL run `pre-workspace-open` hooks before opening and SHALL abort if any fail, and SHALL run `post-workspace-open` hooks after opening, whose failures are logged and do not affect the exit status — the same contract `swm workspace open` observes, so that an ensured workspace is not a different environment from an opened one.

By default the command SHALL print exactly the workspace id followed by a newline, and nothing else, so a caller can capture it directly. With `--json` it SHALL print a single JSON object carrying the workspace id and the pane groups it opened.

#### Scenario: Ensures a workspace and prints its id

- **WHEN** `swm workspace ensure feat-x` is run for a story with two attached projects
- **THEN** `session.OpenWorkspace` is called with both projects' worktree paths, `session.OpenPaneGroup` is called once for each project, and stdout is exactly the returned workspace id followed by a newline

#### Scenario: Nothing is attached and nothing is exec'd

- **WHEN** `swm workspace ensure feat-x` is run with a terminal attached
- **THEN** `session.SwitchTo` is not called, the process is not replaced, and the command exits zero

#### Scenario: Running it again changes nothing

- **WHEN** `swm workspace ensure feat-x` is run against a workspace that is already live with its pane groups open
- **THEN** the command exits zero and prints the same workspace id, having opened nothing new

#### Scenario: A story with no attached projects still yields a workspace

- **WHEN** `swm workspace ensure feat-x` is run for a story with no attached projects
- **THEN** the workspace is opened, no pane group is opened, and the workspace id is printed

#### Scenario: An unknown story is refused, not created

- **WHEN** `swm workspace ensure no-such-story` is run
- **THEN** the command exits non-zero naming the story, no story is created, and no plugin call is made

#### Scenario: No picker is consulted

- **WHEN** `swm workspace ensure` is run with no argument, `$SWM_STORY` unset, a picker plugin configured and a terminal attached
- **THEN** the configured `default_story` is used and the picker plugin is not called

#### Scenario: A failing pre-hook aborts before anything is opened

- **WHEN** a `pre-workspace-open` hook exits non-zero
- **THEN** the command exits non-zero and no workspace or pane group is opened

#### Scenario: A failing post-hook does not fail the command

- **WHEN** a `post-workspace-open` hook exits non-zero after the workspace is open
- **THEN** the failure is logged and the command still exits zero and prints the workspace id

#### Scenario: JSON output

- **WHEN** `swm workspace ensure feat-x --json` is run
- **THEN** stdout is a single JSON object carrying the workspace id and the pane groups opened

#### Scenario: OpenWorkspace error

- **WHEN** `session.OpenWorkspace` returns an error
- **THEN** the command exits non-zero and surfaces the error
