# Design

## Context

The rules are plain Markdown that installers copy or link into `~/.claude/rules`. A common
installer links every top-level `.md` file in `contrib/claude-rules/`, so a new file is picked up
without any config change, but a file in a subdirectory or a renamed file is not.

## Goals / Non-Goals

**Goals:**

- Remove the contradiction between the two rules without making either one depend on the other
  being installed.

**Non-Goals:**

- No automated check of the rule text. The rules are prose for an agent, and a text-matching test
  would only pin wording.
- No change to swm's behaviour or CLI.

## Decisions

- **Defer to the path-resolved story, with a fallback.** The confinement rule uses the story that
  `swm-story-from-path.md` resolved when that rule is installed, and `$SWM_STORY` otherwise. The
  alternative was to merge the two rules into one file. Keeping them separate keeps confinement
  usable on its own, and lets installers that already link both files keep working unchanged.
- **Define `$SWM_STORY` inside the rule as the resolved story.** The from-path rule already tells
  the agent to prefix commands with `export SWM_STORY='<STORY>';`, so the commands in the
  confinement rule work unchanged and only the prose needs to change.
- **A new `agent-rules` capability, not a delta to `project-documentation`.** The rules are
  behaviour shipped to agents, not documentation about swm, so they get their own capability
  that can grow as more rules are added.

## Risks / Trade-offs

- [The rule names its sibling by file name] → The spec requires both file names to stay fixed,
  and the README table lists them.
