## Why

Two unrelated defects, both found by a caller trying to automate swm, both in
the same corner of the CLI.

`swm story list` prints names with cobra's `Println`, which writes to stderr. Its
stdout is empty, so `swm story list | grep -qx feat-x` finds nothing and every
story looks absent. That is the same mistake as `workspace ensure` (#207) and
`config get` (#209): the third command whose whole output goes to the wrong
stream. A steward host agent asked whether a story existed, was told no, and
failed a real assignment while the story's worktree sat on disk.

Separately: nothing a caller opens a pane with can be recognised afterwards. A
caller that opens a pane and wants to find it again — after its own restart, or
to avoid opening a second one — has only `pane_id`, which it must have kept.
`--env` sets the pane's process environment, which `pane list` cannot report and
which dies with the process anyway. `title` is reported but set by whatever runs
in the pane, not by the caller.

agent-mesh solves this for itself by setting a pane-scoped tmux option, which is
why its identity survives `/clear` and a restart in the same pane. Nothing in
swm's own vocabulary offers that, so a caller wanting it has to reach past swm
to the multiplexer — which is exactly what swm exists to make unnecessary.

## What Changes

- `swm story list` writes to stdout.
- `swm pane open` gains `--tag KEY=VALUE`, repeatable: opaque key/value pairs
  the caller attaches to the pane.
- `swm pane list` reports each pane's tags, so a caller can find its own panes
  without having kept their ids.
- Tags live with the pane rather than with the process in it, so they survive a
  program exiting and restarting in that pane, and vanish when the pane closes.
- The session plugin contract carries tags; the tmux provider stores them as
  pane-scoped options, which is where tmux keeps things that belong to a pane.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `workflow-commands`: `story list` writes to stdout; `pane open` takes tags and
  `pane list` reports them.
- `plugin-protocol`: OpenPaneRequest carries tags and Pane reports them.
- `session-tmux`: the tmux provider stores tags as pane-scoped options.

## Impact

- `cmd/swm/internal/cli/story/list.go`: stdout.
- `proto/swm/plugin/v1/session.proto`: `tags` on OpenPaneRequest and on Pane.
  Additive; a provider that ignores them reports none.
- `plugins/session-tmux`: set and read pane options.
- `cmd/swm/internal/cli/pane`: the `--tag` flag and tags in the listing.
- Downstream: steward can mark the pane it opened as its own, which is what it
  needs to avoid opening a second worker beside a human's.
