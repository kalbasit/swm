# Proposal: Create Story on Workspace Open

## Why

`swm workspace open <name>` fails immediately when the named story does not exist,
forcing users to run `swm story create <name>` as a separate step before they can open a
workspace. This is unnecessary friction in the common "start a new story" flow.

## Non-goals

- Auto-creating the story silently without confirmation.
- Changing the behavior of `swm story create`.
- Altering the picker flow (no-arg case where the story is selected interactively).

## What Changes

- **MODIFIED** `swm workspace open <name>`: when `<name>` does not exist in the story
  store, prompt the user interactively ("Story 'name' does not exist. Create it? [y/N]").
  - If confirmed → create the story (equivalent to `swm story create <name>`) then
    continue with the open flow as normal.
  - If declined or stdin is not a TTY → exit non-zero with the existing "story not found"
    error (backward-compatible for scripted use).

## Capabilities

### New Capabilities

_(none)_

### Modified Capabilities

- `workflow-commands` — the `swm workspace open` "Story not found" scenario changes:
  current behavior (immediate non-zero exit) becomes a confirmation prompt that may
  create the story before proceeding.

## Impact

- `cmd/swm` — `workspace open` subcommand handler.
- `openspec/specs/workflow-commands/spec.md` — "Story not found" scenario replaced with
  new "Story not found — user confirms creation" and "Story not found — user declines"
  scenarios; TTY-detection requirement added.
- No proto changes required.
- No new dependencies; TTY detection uses `golang.org/x/term` (already vendored).
