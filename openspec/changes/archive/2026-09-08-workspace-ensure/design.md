## Context

See proposal.md — Why. The relevant shape of the existing code: `open.go` has
two paths, `openWithPicker` and `openAllAttached`. Both end the same way —
`session.SwitchTo`, then `execFn` on the returned argv, replacing the process
with `tmux attach-session`. Everything before that in `openAllAttached` is
exactly what `ensure` needs:

1. derive each attached project's worktree path via `layout.Resolver`
2. `session.OpenWorkspace` with those paths
3. `session.OpenPaneGroup` for a project
4. `post-workspace-open` hooks
5. — and then SwitchTo/exec, which `ensure` stops before

`session.OpenWorkspace` is already "open or attach to": the tmux provider
returns the existing socket when one is live. `OpenPaneGroup` is the same for a
group that already exists. So idempotence is inherited, not built.

## Goals / Non-Goals

**Goals:**

- A caller with no terminal can get a live workspace and capture its id.
- `ensure` and `open` cannot drift in what a workspace *is*, because they share
  the code that makes one.
- `swm workspace open` behaves exactly as it does today.

**Non-Goals:**

- Closing or tearing down a workspace. `swm workspace close` exists.
- Creating stories or worktrees. `swm story create` and `swm story attach` do
  that, and a command that quietly created a story would make the failure mode
  of a typo be "a new story appears".
- Any plugin protocol change. Nothing new is being asked of a session provider.

## Decisions

### A sibling command, not a flag or a tty check

Decided with the operator. `swm pane open` already established the shape:
print the id, exit, say nothing else. `ensure` is that for workspaces.

*Alternative considered:* `swm workspace open --detach`. Rejected — one flag
would have to disable the picker, the story-not-found prompt, the SwitchTo and
the exec together, which is four behaviours behind one word, and `open` would
have two audiences with different contracts.

*Alternative considered:* have `open` detect that stdin is not a tty and behave
differently. Rejected — a human running `swm workspace open x | tee log` would
silently get the daemon path. Behaviour that depends on invisible context is
behaviour nobody can predict from the command line they typed.

### Pane groups for every attached project, not the first

`openAllAttached` opens one pane group, for `st.Projects[0]`, because it needs
somewhere to put the cursor before switching. `ensure` has no cursor to place,
and a caller asking for a story's workspace wants the story's projects in it.

This is not cosmetic. A caller that opens a pane in a specific project's group
— which is what the steward agent does — would find that group absent whenever
its project was not the story's first. One project's work would succeed and
another's would fail, for a reason that has nothing to do with either.

### The shared part is extracted, and it is the part that has no opinion

`openAllAttached` becomes a thin caller over a function that opens the
workspace and the pane groups and returns their ids. That function knows about
worktree paths and the session plugin; it knows nothing about switching,
exec'ing, killing panes or pickers, all of which stay in `open.go`.

The test that matters here is not that the code is shared but that the sharing
is on the right seam: `ensure` must not be able to switch, so switching must
not be inside what it calls.

### Hooks run, and the post-hook contract is copied exactly

An ensured workspace runs the same `pre-workspace-open` and
`post-workspace-open` hooks, with the same asymmetry: a failing pre-hook aborts,
a failing post-hook is logged and ignored. Hooks are how a workspace becomes
the environment the user configured; an ensured workspace that skipped them
would be a second, subtly different kind of workspace, and the difference would
surface as "it works when I open it by hand".

`post-workspace-open` receives `WorkDir` — `open` passes the first project's
worktree path. `ensure` passes the same, so the hook sees what it has always
seen.

### An unknown story is an error, with no prompt

`open` prompts on a tty and errors without one. `ensure` errors, always. The
caller that wants a story created has `swm story create`, which is one more
line and cannot be triggered by a typo.

## Risks / Trade-offs

- **A story with many projects opens many pane groups** → more tmux windows
  than `open` creates up front. This is what the user ends up with anyway after
  navigating the story, and the alternative silently breaks callers whose
  project is not first.
- **Extracting from `openAllAttached` touches a path `open` depends on** →
  covered by `open`'s existing tests; the refactor is only correct if those
  still pass untouched, which is the check to insist on.
- **Two commands to learn** → mitigated by them being obviously different
  verbs. `open` takes you somewhere; `ensure` makes sure something is there.
