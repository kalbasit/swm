## Why

`swm story remove` currently requires the story name as a mandatory positional argument, forcing users to type `swm story remove $SWM_STORY` when they want to remove the story they are currently working in. Since `$SWM_STORY` is already set in every story workspace, the name should default to it so users can simply run `swm story remove` from within a story.

## What Changes

- Make the `<name>` argument to `swm story remove` optional (0 or 1 args).
- When `<name>` is omitted, resolve it from the `$SWM_STORY` environment variable.
- If `<name>` is omitted and `$SWM_STORY` is unset or empty, the command SHALL exit non-zero with a clear error message.
- Update `cmd/swm/README.md` to reflect the new optional argument.

## Non-goals

- Changing the confirmation prompt behaviour.
- Adding env-var fallback to any other story sub-command (`create`, `list`).
- Changing the resolution order when the positional arg IS provided.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `workflow-commands` — the requirement for `swm story remove` changes: `<name>` becomes optional with `$SWM_STORY` as the fallback. A delta spec is needed to update the existing requirement and add new scenarios.

## Impact

- `cmd/swm/internal/cli/story/remove.go` — change `cobra.ExactArgs(1)` to `cobra.MaximumNArgs(1)`; add env-var resolution when no arg is supplied.
- `cmd/swm/internal/cli/story/remove_test.go` — new unit-test scenarios.
- `cmd/swm/tests/integration/integration_test.go` — extend integration tests.
- `cmd/swm/README.md` — update usage line for `swm story remove`.
- `openspec/specs/workflow-commands/spec.md` (via delta) — updated requirement + new scenarios.
- No proto changes. No new plugins. No config changes.
