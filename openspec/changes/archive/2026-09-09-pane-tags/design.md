## Context

See proposal.md — Why. Two independent things in one change because they are
the same size, the same corner of the CLI, and would otherwise be two releases
a downstream caller has to wait through in sequence.

`swm story list` already specifies stdout. The spec is right and the code is
wrong, so nothing about the contract changes: the requirement gains a scenario
that would have caught it, and the implementation is corrected to match what it
already promised.

Tags are new. The relevant existing shape: `OpenPaneRequest` carries
`env`, a `map<string, string>` the tmux provider applies to the pane's process,
and `Pane` carries descriptive fields the provider fills in best-effort.

## Goals / Non-Goals

**Goals:**

- A caller can recognise a pane it opened, without having kept the pane id.
- That recognition survives the program in the pane exiting and being replaced,
  which is the case `--env` cannot serve.
- `swm story list` is usable in a pipeline.

**Non-Goals:**

- Interpreting tags. They are opaque to swm: it stores them and reports them.
  A tag that means "steward owns this pane" means that to steward and nothing
  to swm.
- Querying by tag. `pane list` reports them and a caller filters; a `--tag`
  filter is easy to add later and nothing needs it yet.
- Tags on a pane group or a workspace. Nothing has asked, and each would be a
  different lifetime to reason about.

## Decisions

### Tags live on the pane, not in the process

Stored as pane-scoped tmux options, which is exactly where agent-mesh puts the
agent id that survives `/clear` and a restart in the same pane. tmux keeps
pane options with the pane, so they outlive the process and vanish with the
pane — which is the lifetime a caller marking "I opened this" actually wants.

*Alternative considered:* `--env`, which already exists. Rejected: the
environment belongs to the process, so it dies when the program exits, and
`ListPanes` cannot report it — a caller would have to read `/proc` for a pid it
would first have to find, which is precisely the reaching-past-swm this is
meant to remove.

*Alternative considered:* the pane title. Rejected: the title is set by whatever
runs in the pane, so a caller's mark would be overwritten by the first program
that sets one, and it is a single string where a caller wants keys.

### Opaque to swm

swm stores and returns tags and never reads them. The moment swm understands a
tag it acquires an opinion about which callers may set it and what happens when
two disagree — and there is no such conflict to arbitrate.

This is why they are absent rather than empty when unset: a caller checking for
its own tag should distinguish "this pane carries none" from "this pane carries
an empty one", and the difference is free if nothing invents a default.

### `--tag` parses exactly as `--env`

Split on the first `=`, further `=` characters kept, an entry without one
refused before any plugin call. Two flags on one command that both take
`KEY=VALUE` and disagreed about parsing would be a trap, and the existing rule
is already the right one.

## Risks / Trade-offs

- **Pane options are a tmux concept** → the contract speaks of tags and says
  nothing about tmux; a provider without an equivalent reports none, exactly as
  it already may for a title.
- **Two unrelated fixes in one change** → they land together or not at all,
  which is worth one review and one release rather than two of each.
- **A caller could put something large in a tag** → tmux options are not a
  store, and nothing here validates size. It is a mark, and the contract says
  so.
