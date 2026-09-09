## Context

See proposal.md — Why. Everything this command reports already exists in
process: `coreStory.Store.Get` returns the story with its branch name and
attached projects, and `layout.Resolver.WorktreePath` composes a project's path
for a story, including the special case where the default story resolves to the
canonical clone rather than a worktree. Nothing new has to be computed; the
facts simply have no way out.

## Goals / Non-Goals

**Goals:**

- A caller outside the process can learn a story's branch name and where its
  projects live, without reimplementing how swm derives either.
- What `show` reports and what `workspace ensure` opens agree by construction,
  because both ask the same resolver.

**Non-Goals:**

- Reporting anything about a running session. `swm pane list` and
  `swm workspace list` answer that; a story exists whether or not a workspace
  is open, and conflating the two would make `show` fail for a story that is
  merely closed.
- Reporting the branch a worktree currently has checked out. That is a
  different fact, it changes under anyone who switches branches, and `git` in
  the worktree already answers it for a caller who wants it.
- Listing every story. `swm story list` does that.

## Decisions

### A read of swm's records, not of the world

`show` reads the story store and the layout resolver, and consults no plugin.
This is what makes it usable from a daemon and what makes it total: a story
with no workspace open, or on a machine where the multiplexer is not running,
still reports in full.

It also means a reported path is where the worktree *should* be, not proof that
it is there. That is the right answer for the caller this exists for: steward
asks swm to materialise the story first and then asks what it materialised, so
a path that does not exist means the attach failed, which is already an error
on its own.

### The default story is not special-cased here

`WorktreePath` already returns the canonical clone for the default story,
because that story has no separate worktree. Reimplementing that condition in
this command would be a second place for it to be wrong. The command asks the
resolver and reports the answer.

### Project keys, not display spellings

A project is reported as `<host>/<segments...>` — `github.com/kalbasit/swm`.
That is how a project is named everywhere except inside a multiplexer, where
`swm pane list` reports a pane group id with dots replaced. A caller matching
`show` output against what it asked for needs the former; anything needing the
latter is addressing a pane and should be reading pane output.

### `--json` alongside human output, matching the pane commands

The default is readable; `--json` is a single object. This follows what
`swm pane list` and `workspace ensure` already do, so a script does not have to
learn a different convention per command.

## Risks / Trade-offs

- **A reported path may not exist on disk** → documented above and in the
  command's help. The alternative — stat every path before reporting — makes a
  read of records into a read of the filesystem, and turns a story that is
  merely not yet attached into an error.
- **Another command to keep in step with the layout** → mitigated by not
  reimplementing the layout: the resolver is the single source, and a change
  there changes this command's answer automatically.
