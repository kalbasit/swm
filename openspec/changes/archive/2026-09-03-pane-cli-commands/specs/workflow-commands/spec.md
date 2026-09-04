## ADDED Requirements

### Requirement: CLI exit status is carried by the error

The `swm` binary SHALL exit `0` on success and non-zero on failure. A command
that needs a specific non-zero status SHALL return an error that reports it,
rather than terminating the process itself: production code outside `main` MUST
NOT call `os.Exit`.

`cmd/swm/internal/exitcode` SHALL provide that channel. An error anywhere in the
returned error chain that implements `ExitCode() int` SHALL determine the
process exit status. Any other non-nil error SHALL produce exit status `1`.
`main` SHALL print the error to stderr and exit with the resolved status.

Exit status `2` SHALL NOT be assigned, so that it remains available for the
conventional usage-error meaning.

#### Scenario: Plain error exits 1

- **WHEN** a command returns an error that carries no exit code
- **THEN** `swm` prints the error to stderr and exits `1`

#### Scenario: Error carrying a code exits with that code

- **WHEN** a command returns an error carrying exit code `3`
- **THEN** `swm` prints the error to stderr and exits `3`

#### Scenario: Wrapped error still carries its code

- **WHEN** an error carrying exit code `3` is wrapped with additional context
- **THEN** the resolved exit status is still `3`

### Requirement: swm pane command group

`swm pane` SHALL be a command group exposing the `Session` capability's pane
primitives, with subcommands `open`, `list`, `send`, and `close`. Each
subcommand SHALL be a thin wrapper over the corresponding RPC — `OpenPane`,
`ListPanes`, `SendText`, `ClosePane` — and SHALL NOT interpret, parse, or
reconstruct a `pane_id`, which is an opaque provider handle.

Every subcommand SHALL obtain the session plugin through the plugin manager and
SHALL exit non-zero with a descriptive error when no session plugin is
configured or the resolved plugin is not a `SessionClient`.

Because a `pane_id` is meaningful only within the workspace that minted it,
every subcommand that accepts one SHALL also require `--workspace`.

#### Scenario: Session plugin absent

- **WHEN** any `swm pane` subcommand is run and no session plugin is configured
- **THEN** the command exits non-zero with a descriptive error and no pane
  operation is attempted

### Requirement: swm pane open

`swm pane open --workspace <id> --pane-group <id> [--cwd <dir>] [--env K=V]... [--json] [-- <argv>...]`
SHALL call `session.OpenPane` and report the resulting pane.

- `--workspace` (`-w`) and `--pane-group` (`-g`) are required; omitting either
  SHALL exit non-zero before any plugin call.
- Positional arguments SHALL be passed as `argv` verbatim, in order, with no
  re-splitting. No positional arguments means an empty `argv`, which the
  contract defines as the provider's default shell.
- `--cwd` SHALL be passed as the request's `cwd`.
- `--env` MAY be repeated. Each value SHALL be split on its first `=` into key
  and value; a value containing further `=` characters keeps them. An entry with
  no `=`, or with an empty key, SHALL exit non-zero before any plugin call.
- By default the command SHALL print exactly the returned `pane_id` followed by
  a newline, and nothing else, so that a caller can capture it directly.
- With `--json` the command SHALL print the pane as a single JSON object.

#### Scenario: Opens a pane and prints its id

- **WHEN** `swm pane open -w /run/swm/feat-x.sock -g github.com/kalbasit/swm -- nvim main.go` is run
- **THEN** `session.OpenPane` is called with that workspace, that pane group, and `argv=["nvim","main.go"]`, and stdout is exactly the returned pane id followed by a newline

#### Scenario: No argv opens the default shell

- **WHEN** `swm pane open -w W -g G` is run with no positional arguments
- **THEN** `session.OpenPane` is called with an empty `argv`

#### Scenario: Argument containing spaces is not re-split

- **WHEN** `swm pane open -w W -g G -- sh -c "echo hello world"` is run
- **THEN** `argv` has exactly three elements and the third is `echo hello world`

#### Scenario: Environment entries are passed through

- **WHEN** `swm pane open -w W -g G --env FOO=bar --env EQ=a=b` is run
- **THEN** the request's `env` map contains `FOO=bar` and `EQ` mapped to `a=b`

#### Scenario: Malformed environment entry

- **WHEN** `swm pane open -w W -g G --env FOO` is run
- **THEN** the command exits non-zero with an error naming the malformed entry and no plugin call is made

#### Scenario: JSON output

- **WHEN** `swm pane open -w W -g G --json` is run
- **THEN** stdout is a single JSON object carrying the pane fields

#### Scenario: Missing required flag

- **WHEN** `swm pane open -w W` is run without `--pane-group`
- **THEN** the command exits non-zero and no plugin call is made

#### Scenario: OpenPane error

- **WHEN** `session.OpenPane` returns an error
- **THEN** the command exits non-zero and surfaces the error

### Requirement: swm pane list

`swm pane list [--workspace <id>] [--pane-group <id>] [--json]` SHALL call
`session.ListPanes` and print every pane the stream yields, preserving the order
the plugin produced.

Both filters are optional and default to unset. An unset `--workspace`
enumerates every live workspace and an unset `--pane-group` enumerates every
pane group within them, exactly as `ListPanesRequest` specifies.

With `--json` the command SHALL print a JSON array of pane objects, and SHALL
print `[]` when the stream is empty — never `null`. Without `--json` it SHALL
print a human-readable table with a header row, and print nothing when the
stream is empty.

#### Scenario: Lists every pane on the host

- **WHEN** `swm pane list --json` is run with no filters
- **THEN** `session.ListPanes` is called with both filters empty and stdout is a JSON array of every pane returned

#### Scenario: Filters are forwarded

- **WHEN** `swm pane list -w W -g G` is run
- **THEN** `session.ListPanes` is called with `workspace_id=W` and `pane_group_id=G`

#### Scenario: No panes in JSON mode

- **WHEN** `swm pane list --json` is run and the stream yields no panes
- **THEN** stdout is `[]` followed by a newline and the command exits zero

#### Scenario: No panes in table mode

- **WHEN** `swm pane list` is run and the stream yields no panes
- **THEN** stdout is empty and the command exits zero

#### Scenario: Focused panes are visible

- **WHEN** `swm pane list --json` is run and one pane is reported as focused
- **THEN** that pane's `focused` field is `true` and every other pane's is `false`

#### Scenario: Stream error

- **WHEN** the `ListPanes` stream returns an error mid-way
- **THEN** the command exits non-zero and surfaces the error

### Requirement: swm pane send

`swm pane send --workspace <id> --pane <id> [--submit] [--delay-ms <n>] [--allow-focused] [<text>]`
SHALL call `session.SendText` with the given text.

- `--workspace` (`-w`) and `--pane` (`-p`) are required.
- The optional positional argument is the text, delivered literally. No part of
  it is interpreted as a key name.
- `--submit` appends the provider's submit key after the text.
- `--delay-ms` sets `delay_ms`, observed by the plugin before delivery.
- `--allow-focused` sets `allow_focused`. The CLI SHALL NOT set it on the
  caller's behalf under any other circumstance.
- Omitting the text argument without `--submit` SHALL exit non-zero with an
  error naming `--submit`, before any plugin call, because there is nothing to
  deliver.
- On success the command SHALL print nothing and exit zero.

#### Scenario: Sends text

- **WHEN** `swm pane send -w W -p %4 "hello"` is run
- **THEN** `session.SendText` is called with that workspace, that pane, `text="hello"`, `submit=false`, and `allow_focused=false`, and the command exits zero with no output

#### Scenario: Submit key only

- **WHEN** `swm pane send -w W -p %4 --submit` is run with no text argument
- **THEN** `session.SendText` is called with empty text and `submit=true`

#### Scenario: Nothing to send

- **WHEN** `swm pane send -w W -p %4` is run with no text argument and without `--submit`
- **THEN** the command exits non-zero with an error naming `--submit` and no plugin call is made

#### Scenario: Delay is forwarded

- **WHEN** `swm pane send -w W -p %4 --delay-ms 250 --submit` is run
- **THEN** the request carries `delay_ms=250`

#### Scenario: Text is not interpreted

- **WHEN** `swm pane send -w W -p %4 "Enter"` is run
- **THEN** the request carries the five-character text `Enter` and `submit=false`

### Requirement: swm pane send reports a focused-pane refusal distinctly

When `session.SendText` fails with gRPC status `FAILED_PRECONDITION` — the
status the `Session` contract assigns to delivery into a pane the provider
reports as focused — `swm pane send` SHALL exit with status `3` and print an
error that states the pane is focused and names `--allow-focused` as the
override.

Every other failure of `swm pane send` SHALL exit `1`. A caller SHALL therefore
be able to distinguish a focused-pane refusal from any other error by exit
status alone, without parsing stderr.

Exit status `3` SHALL NOT be reused by any other `swm pane` failure.

#### Scenario: Focused pane refuses with exit 3

- **WHEN** `swm pane send -w W -p %4 "hello"` targets a pane an attached client is focused on and `--allow-focused` is not set
- **THEN** the plugin returns `FAILED_PRECONDITION`, the command exits `3`, and the error text names `--allow-focused`

#### Scenario: --allow-focused delivers into a focused pane

- **WHEN** the same command is run with `--allow-focused`
- **THEN** `allow_focused` is set on the request, the plugin delivers the text, and the command exits zero

#### Scenario: Other failures stay on exit 1

- **WHEN** `swm pane send` targets a pane that does not exist and the plugin returns `NOT_FOUND`
- **THEN** the command exits `1`

### Requirement: swm pane close

`swm pane close --workspace <id> --pane <id>` SHALL call `session.ClosePane`
and print nothing on success.

- `--workspace` (`-w`) and `--pane` (`-p`) are required.
- Closing a pane that no longer exists is not an error: the contract defines
  `ClosePane` as idempotent, and the CLI SHALL surface that unchanged.

#### Scenario: Closes a pane

- **WHEN** `swm pane close -w W -p %4` is run
- **THEN** `session.ClosePane` is called with that workspace and pane, and the command exits zero with no output

#### Scenario: Already-closed pane succeeds

- **WHEN** `swm pane close -w W -p %4` is run for a pane whose program already exited
- **THEN** the command exits zero

#### Scenario: ClosePane error

- **WHEN** `session.ClosePane` returns an error
- **THEN** the command exits non-zero and surfaces the error

### Requirement: Pane JSON encoding

The JSON emitted by `swm pane open --json` and `swm pane list --json` SHALL use
snake_case keys matching the proto field names, and SHALL always include every
field, even when a best-effort descriptive field is empty:

```json
{
  "pane_id": "%4",
  "pane_group_id": "github.com/kalbasit/swm",
  "workspace_id": "/run/user/1000/swm/tmux/feat-x.sock",
  "title": "nvim",
  "current_command": "nvim",
  "current_path": "/home/user/code/stories/feat-x/github.com/kalbasit/swm",
  "focused": false
}
```

#### Scenario: Empty descriptive fields are present

- **WHEN** a plugin returns a pane with no title, command, or path
- **THEN** the JSON object still contains `title`, `current_command`, and `current_path` as empty strings

#### Scenario: Keys are snake_case

- **WHEN** any pane JSON is emitted
- **THEN** the keys are `pane_id`, `pane_group_id`, `workspace_id`, `title`, `current_command`, `current_path`, and `focused`
