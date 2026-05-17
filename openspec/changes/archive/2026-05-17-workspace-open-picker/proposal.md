## Why

`swm workspace open` without arguments opens the default story directly, forcing users to type a
story name to reach any other story. As story counts grow, discoverability suffers and the
friction encourages staying in the default story longer than intended.

## What Changes

- **New story-picker flow in `workspace open`**: when invoked without a story-name argument, a
  picker plugin is available, and a TTY is present, show a story list instead of opening the
  default story directly. Selecting an entry then proceeds to the existing project-within-story
  picker flow unchanged.
- **Story entry display format**: each picker entry shows story name, branch name in parentheses
  only when it differs from the story name, age since creation (rounded up: minutes → hours →
  days → weeks → months → years), and attached projects joined with ` · `.
- **`_default` label**: the default story is listed as `_default (main repo)` to distinguish it
  from feature stories.
- **Terminal-width-aware truncation**: the host detects terminal width via `/dev/tty`,
  falling back to `$COLUMNS`, then to 120 columns. Display strings are truncated before being
  streamed to the picker plugin. Truncation priority (right to left): projects list → branch
  name → story name.
- **New internal utilities**: rounded-up age formatting; terminal-width detection.

## Capabilities

### New Capabilities

None — no new plugin capability surface is introduced.

### Modified Capabilities

- `picker`: the host now calls the picker plugin twice in sequence when no story name is given
  (story selection first, then project selection). No proto changes required; `PickItem` (Key +
  Display) is already generic enough to carry story entries.

## Non-goals

- No changes to the picker plugin gRPC protocol or `PickItem`/`PickResult` proto messages.
- No multi-select; exactly one story is chosen per invocation.
- No changes to `swm workspace open <story-name>` (explicit arg skips the story picker).
- No changes to the project-within-story picker that runs after story selection.
- No UI changes to `swm workspace list`.

## Impact

- **`cmd/swm/internal/cli/workspace/open.go`**: primary change — new story-picker path before
  existing project-picker path.
- **New internal package(s)**: age formatting (rounded-up duration), terminal-width detection.
- **No proto changes**, no plugin binary changes, no config schema changes.
