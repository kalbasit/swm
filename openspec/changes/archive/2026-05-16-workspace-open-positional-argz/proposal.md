# Proposal: workspace-open-positional-argz

## Why

`swm workspace open --story <name>` is unnecessarily verbose for the most common operation in swm. Positional arguments are the idiomatic CLI convention for primary operands, and the `--story` flag adds friction compared to `swm workspace open swm-use-gh`.

## What Changes

- **BREAKING**: `swm workspace open <story-name>` replaces `swm workspace open --story <story-name>` as the canonical form
- Remove the `--story` flag from `workspace open`
- Positional story name remains optional; fallback order is unchanged: positional arg → `$SWM_STORY` → `cfg.DefaultStory`

## Non-goals

- Changing fallback behaviour (env var and default story remain)
- Modifying any other command's argument style

## Capabilities

### New Capabilities

_(none)_

### Modified Capabilities

- `workflow-commands`: `workspace open` switches from a named flag to a positional argument for story name — spec-level CLI contract change

## Impact

- `cmd/swm/internal/cli/workspace/open.go` — replace `StringVar` flag with `cobra.Args` positional arg parsing
- Integration tests in `cmd/swm/tests/integration/` that invoke `workspace open --story` must be updated
- Any documentation or examples using `--story` flag must be updated
