# Proposal

## Why

swm ships Claude Code rules in `contrib/claude-rules/` so that an agent session behaves the same
under swm as under maktab, its sibling story manager. A rule-by-rule comparison shows the two sides
already agree, with one flaw both share: `swm-story-confinement.md` still tells the agent to detect
the story by reading `$SWM_STORY`, while `swm-story-from-path.md` says never to trust that variable.
An agent with both rules installed gets two contradictory instructions. Nothing specifies what the
contrib rules must contain, so this kind of drift goes unnoticed.

## What Changes

- `swm-story-confinement.md` detects the story from the story that `swm-story-from-path.md`
  resolved from the working directory when that rule is installed, and falls back to
  `$SWM_STORY` only when it is not. Inside the rule, `$SWM_STORY` means that resolved story.
- A new `agent-rules` spec records the contract for `contrib/claude-rules/`: which rules ship,
  where they live, how they are worded, and that the README lists each one.
- Docs only. No code, CLI, or config change.

## Capabilities

### New Capabilities

- `agent-rules`: the Claude Code rules swm ships under `contrib/claude-rules/`. Covers their
  layout, their swm-only wording, the story-resolution order between the two rules, and their
  listing in the root README.

### Modified Capabilities

(none)

## Impact

- `contrib/claude-rules/swm-story-confinement.md`: the story-detection paragraph changes.
- `openspec/specs/agent-rules/spec.md`: new capability spec.
- Anyone who links every `.md` file from `contrib/claude-rules/` into `~/.claude/rules` picks up
  the change the next time they update swm. They don't need to change their setup.
