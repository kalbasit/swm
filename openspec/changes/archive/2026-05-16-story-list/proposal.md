# Proposal: swm story list

## Why

Users can create and remove stories but have no way to enumerate them from the
CLI — they must inspect the JSON store directly. A `swm story list` command
closes this gap and makes the story lifecycle fully introspectable from the
terminal.

## What Changes

- Add `swm story list` sub-command under `swm story`.
- The command calls `Store.List` (already implemented in `story-store`) and
  prints each story name, one per line, to stdout.
- No flags required for the initial implementation.
- No proto changes; no new plugins involved.

## Non-goals

- Filtering, sorting, or formatting flags (e.g. `--json`, `--active`).
- Displaying story metadata beyond the name (projects, branch, timestamps).
- Interactive/picker mode.

## Capabilities

### New Capabilities

_(none — `Store.List` already exists; only the CLI surface is new)_

### Modified Capabilities

- **workflow-commands** — add `swm story list` requirement. The delta spec
  will document the command contract, output format, and scenarios (empty
  store, single story, multiple stories).

## Impact

- New file: `cmd/swm/internal/cli/story/list.go` (and `list_test.go`).
- `cmd/swm/internal/cli/root.go`: register the new sub-command.
- `openspec/specs/workflow-commands/spec.md`: delta with `swm story list`
  requirement.
- No dependency changes.
