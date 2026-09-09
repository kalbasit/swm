## Why

`swm workspace open` is built for a human at a terminal, and it ends by
replacing its own process: `session.SwitchTo` returns an argv and the command
calls `syscall.Exec` on `tmux attach-session`. Before that it may show a story
picker and may prompt on stdin to create a missing story. None of that survives
being run from a daemon — the process is either replaced or it fails, and
either way the caller does not come back.

There is no other route to a live workspace. `swm pane open` requires a
workspace id that already exists, and a story materialised by
`swm story attach` has worktrees on disk but no session: `laio-plugin` on this
machine is exactly that today — three worktrees, no socket.

So anything automating swm can create a story, create its worktrees, and then
stop, one step short of anywhere to run something. The steward host agent is
the case in hand: it receives work for a story, materialises it, and cannot
open a pane in it.

## What Changes

- A new command, `swm workspace ensure [<story-name>]`, that makes a story's
  workspace and its pane groups exist and then exits, printing the workspace id
  on stdout and nothing else.
- It never attaches, never execs, never picks and never prompts — with or
  without a terminal. A command whose behaviour depends on whether stdin
  happens to be a tty is a command nobody can reason about from a script.
- It opens a pane group for **every** project attached to the story, not only
  the first. `open` opens the first because it is about to switch you there;
  `ensure` has no cursor to place and a caller asking for a workspace wants the
  story's projects present, not one of them.
- It runs the same `pre-workspace-open` and `post-workspace-open` hooks as
  `open`, so an ensured workspace is not a subtly different environment from an
  opened one.
- `swm workspace open` is unchanged.

## Capabilities

### New Capabilities

None. This is a new command in a command surface that already exists.

### Modified Capabilities

- `workflow-commands`: the new `swm workspace ensure` command — what it opens,
  what it prints, what it refuses, and what it deliberately does not do.
- `shell-completion`: story-name completion for the new command's positional
  argument, matching what `workspace open` and `story remove` already offer.

## Impact

- `cmd/swm/internal/cli/workspace`: a new `ensure.go` alongside `open.go`, plus
  `materialize.go` holding the part of `openAllAttached` that precedes
  `SwitchTo` — extracted so the two commands cannot drift, and so `ensure` is
  structurally unable to switch. Story-name completion moves to `completion.go`
  and is shared rather than duplicated.
- No plugin protocol change. `session.OpenWorkspace` and
  `session.OpenPaneGroup` already do what is needed and are already idempotent
  in the tmux provider.
- `README` for the host CLI, and shell completion registration.
- Downstream: the steward host agent's `story-api` change depends on this
  command existing, and on a released swm containing it.
