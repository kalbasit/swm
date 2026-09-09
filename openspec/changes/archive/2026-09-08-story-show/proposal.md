## Why

`swm story list` prints story names and nothing else. Nothing reports a story's
branch name, and nothing reports where its projects are materialised. Both
facts exist — they are in the story's own JSON and derivable by the layout
resolver — and neither is reachable from outside the process.

That leaves a caller automating swm with only bad options. It can read the
branch with `git rev-parse` in a worktree, which answers a different question:
the branch checked out at that moment, which stops being the story's branch the
first time anyone switches, and switching is the normal course of work rather
than an edge case. Or it can compose worktree paths from `code_root` itself,
which puts swm's on-disk layout inside the caller, where neither project's
tests would notice it drifting — swm would keep passing because it never
promised the shape, and the caller would keep passing because it never talks to
a real swm.

The steward host agent is the case in hand. It materialises a story for work
placed on its machine and must report back what it created; today it can create
the thing and not describe it.

## What Changes

- A new command, `swm story show [<story-name>] [--json]`, reporting one story:
  its name, its branch name, when it was created, and each attached project
  with the worktree path it resolves to on this host.
- Paths are resolved through the same layout resolver every other command uses,
  so what `show` reports and what `workspace ensure` opens cannot disagree.
- An unknown story exits non-zero naming it, like `workspace ensure`, rather
  than printing an empty record.

## Capabilities

### New Capabilities

None. This is a new command in a command surface that already exists.

### Modified Capabilities

- `workflow-commands`: the new `swm story show` command — what it reports, in
  what form, and what it refuses.
- `shell-completion`: story-name completion for its positional argument,
  matching `story remove`, `workspace open` and `workspace ensure`.

## Impact

- `cmd/swm/internal/cli/story`: a new `show.go`. No store or resolver change:
  `coreStory.Store.Get` already returns the branch name and the attached
  projects, and `layout.Resolver.WorktreePath` already composes the path.
- No plugin protocol change and no session plugin involvement. This reads
  swm's own records; it does not touch a multiplexer.
- Downstream: steward's `story-api` change is blocked on this command existing
  in a released swm.
