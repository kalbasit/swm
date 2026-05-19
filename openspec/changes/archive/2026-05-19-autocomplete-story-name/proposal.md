## Why

Commands that accept a story name positional argument (`workspace open`, `story remove`) require users to type the name verbatim with no shell completion support. This forces users to remember exact story names or run `swm story list` first, breaking the fast-switching workflow swm is designed for.

## What Changes

- Extend shell completion for `swm workspace open [story]` to complete available story names from the story store.
- Extend shell completion for `swm story remove [story]` to complete available story names from the story store.

## Capabilities

### New Capabilities

<!-- None -->

### Modified Capabilities

- `shell-completion` — add requirements for dynamic story-name completions on commands that accept a story name argument.

## Non-goals

- Completing story names for `swm story create` (creation implies a new, previously unknown name).
- Adding story-name completion to plugin-facing or internal APIs.
- Changing how the interactive story picker behaves when no argument is provided to `workspace open`.

## Impact

- Capability surface: none (shell completion is a CLI surface, not session/vcs/forge/picker/hook).
- No proto changes required.
- Affected code: command definitions for `workspace open` and `story remove` — each needs a `ValidArgsFunction` (cobra) that queries the story store and returns story names as completion candidates.
- No new dependencies.
