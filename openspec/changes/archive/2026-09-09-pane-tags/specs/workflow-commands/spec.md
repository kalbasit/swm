## MODIFIED Requirements

### Requirement: swm story list
`swm story list` SHALL print all story names to stdout, one per line, in
lexical order. The command takes no arguments and no flags. On success it exits
zero. If the store cannot be read it exits non-zero and prints an error to
stderr.

Stdout is the contract, not a description of where the text happens to appear.
A caller pipes this command or captures it, and a name printed to stderr leaves
that caller reading an empty list and concluding no story exists. Verifying it
requires capturing the process's own stdout: a test that sets the command's out
writer cannot tell the streams apart, because cobra's Print family resolves to
that writer once it is set.

#### Scenario: Single story (default only)
- **WHEN** `swm story list` is run and only the `_default` story exists
- **THEN** the command exits zero and prints exactly `_default` to stdout

#### Scenario: Multiple stories
- **WHEN** `swm story list` is run and stories `alpha`, `beta`, and `_default` exist
- **THEN** the command exits zero and prints the names in lexical order, one per line

#### Scenario: Store error
- **WHEN** `swm story list` is run and `Store.List` returns an error
- **THEN** the command exits non-zero and prints a human-readable error message

#### Scenario: The names survive being captured
- **WHEN** `swm story list` is run with its stdout captured and its stderr discarded
- **THEN** the captured text contains the story names

### Requirement: swm pane open

`swm pane open --workspace <id> --pane-group <id> [--cwd <dir>] [--env K=V]... [--tag K=V]... [--json] [-- <argv>...]`
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
- `--tag` MAY be repeated, and SHALL be split the same way as `--env`. Tags are
  attached to the pane rather than to the program in it, so a caller can
  recognise a pane it opened without having kept the pane id, and after the
  program in that pane has exited and been replaced.
- By default the command SHALL print exactly the returned `pane_id` followed by
  a newline, and nothing else, so that a caller can capture it directly.
- With `--json` the command SHALL print the pane as a single JSON object.

#### Scenario: Opens a pane and prints its id

- **WHEN** `swm pane open -w /run/swm/feat-x.sock -g github.com/kalbasit/swm -- nvim main.go` is run
- **THEN** `session.OpenPane` is called with that workspace, that pane group, and `argv=["nvim","main.go"]`, and stdout is exactly the returned pane id followed by a newline

#### Scenario: Tags are passed through

- **WHEN** `swm pane open -w W -g G --tag owner=steward --tag item=8f7e` is run
- **THEN** the request's tags carry `owner=steward` and `item=8f7e`

#### Scenario: A malformed tag opens nothing

- **WHEN** `swm pane open -w W -g G --tag owner` is run
- **THEN** the command exits non-zero naming the malformed entry and no plugin call is made

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

Each pane's tags SHALL be reported in the JSON form, so a caller that tagged a
pane can find it again. A pane with no tags SHALL report none rather than an
empty object that a caller must distinguish from absence.

#### Scenario: Lists every pane on the host

- **WHEN** `swm pane list --json` is run with no filters
- **THEN** `session.ListPanes` is called with both filters empty and stdout is a JSON array of every pane returned

#### Scenario: Tags are reported

- **WHEN** `swm pane list --json` is run and a pane carries tags
- **THEN** that pane's object carries those tags

#### Scenario: A pane with no tags reports none

- **WHEN** `swm pane list --json` is run and a pane carries no tags
- **THEN** that pane's object omits tags rather than carrying an empty one

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
