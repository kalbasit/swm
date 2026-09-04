## Context

`Session` today speaks two nouns: Workspace (a tmux socket) and PaneGroup (a
tmux session). Panes exist only as an implementation detail of layout
application. The one place text is ever delivered to a pane is
`plugins/session-tmux/internal/layout/apply.go`, from the layout TOML's
`commands`, at pane-group-creation time. Nothing reaches a pane at runtime.

Constraint that drives everything here: swm owns all multiplexer interaction, so
a caller that needs to run and drive a program in a pane must be able to do it
through the plugin contract, and the contract must stay meaningful if the
provider becomes zellij or wezterm.

## Goals / Non-Goals

**Goals**

- Four primitives that any multiplexer can implement: open a pane running argv,
  enumerate panes, deliver text to a pane, close a pane.
- `ListPanes` good enough to answer "what is running on this host" in one call.
- A default that makes the dangerous case (typing into a pane a human is using)
  require an explicit decision.

**Non-Goals**

- Pane geometry (split direction, size, focus). Layout config owns that.
- Output capture, attach/detach, process supervision.
- Any notion of what a pane is *for*. No "agent", no "job".
- Any CLI surface.

## Decisions

### Vocabulary: pane, not send-keys, not agent

`SendText` is named for what it does to the pane, not for the tmux command that
implements it. `send-keys` is tmux's word; zellij's equivalent is
`write-chars`/`write`, and wezterm's is `send-text`. Likewise nothing in the
contract mentions agents: swm has no reason to know whether the pane holds an AI
coding agent, a build, or a shell. These primitives are useful to anything that
needs a pane.

### `pane_id` is opaque, and this is precedent, not novelty

`SwitchToRequest.close_origin_pane_id` is already documented in `session.proto`
as "the multiplexer-specific pane reference (e.g. `$TMUX_PANE` for
session-tmux)", and `killOriginPane` deliberately passes it to tmux unescaped
with the comment that a `%N` pane ID "is already unambiguous and must not be
escaped with exactTarget". So an opaque, provider-minted pane handle already
crosses this boundary in both directions. Formalising it as `Pane.pane_id` adds
no new class of value; it gives the existing one a message to live in.

A handle is only meaningful within its workspace, so every request that carries
a `pane_id` also carries a `workspace_id`. tmux pane IDs are unique per server
and a server is per-workspace, so there is no global namespace to appeal to.

### `Pane` carries description, not just identity

The task framing asks for `Pane{pane_id, pane_group_id, workspace_id}`. Those
three are kept exactly, but a `ListPanes` that returns only opaque handles
cannot answer the question `ListPanes` exists to answer — the caller would get
back identifiers it is forbidden to parse and nothing else. So `Pane` also
carries `title`, `current_command`, `current_path`, and `focused`, all
provider-reported and documented as best-effort. They are description, never
identity.

`focused` is on `Pane` rather than being a separate RPC because it is the same
provider query that gates `SendText`; exposing it means a caller can see in
advance which panes `SendText` will refuse.

### The human-in-the-pane hazard: refuse by default

Text delivered to a pane is indistinguishable from typing. If a human is at a
prompt — a `y/N` confirmation, a rebase conflict, a password read — injected
text answers it. This is not hypothetical for the intended use: a pane running
an interactive program is exactly the pane a human is most likely to be sitting
in.

swm cannot know whether a human is *about to* type. It can know something
narrower and cheap: whether the pane is the one an attached client's keystrokes
currently reach — `session_attached` non-zero, `window_active`, `pane_active`.
That is a good proxy for "the human's cursor is here".

Decision: **`SendText` refuses a focused pane with `FAILED_PRECONDITION` unless
the caller sets `allow_focused`.** Rejected alternatives:

- *Docs-only warning.* The failure mode is silent and destructive, and the
  warning is read once while the call is made a thousand times.
- *Always refuse.* Legitimate callers exist — a human deliberately dictating
  into their own focused pane.
- *Refuse-by-default added later.* Turning a permissive default into a
  restrictive one is a breaking change for every existing caller. Free now,
  expensive later. This is the strongest reason to decide it in this change.

The check is racy by construction and is documented as such: it is a guard
against the common accident, not a security boundary. `allow_focused` is a
neutral name — every multiplexer has a focused pane; none of them has to call it
that internally.

### `delay_ms` mirrors `pane_cmd_delay`

`layout.sendKeys` waits `pane_cmd_delay` milliseconds *before* each `send-keys`,
via a `select` on `ctx.Done()` and `time.After` so a cancelled context aborts.
That knob exists because someone hit the race where a program has been spawned
but has not yet installed its input handler, and early keystrokes are lost.

`SendTextRequest.delay_ms` reuses that semantic exactly — wait, then deliver,
context-aware — so the two knobs cannot drift into meaning different things.

There is deliberately no second delay between the text and the submit key. A
caller that needs one issues two calls: `{text: "...", submit: false}` then
`{text: "", submit: true, delay_ms: N}`. Empty text with `submit: true` is
explicitly valid; empty text with `submit: false` is `INVALID_ARGUMENT`.

### Literal delivery is a correctness requirement, not a nicety

`layout.sendKeys` calls `send-keys -t <pane> <cmd> Enter` without `-l`. For
shell command lines that is fine. For arbitrary text it is not: tmux parses
unflagged arguments as key names, so `Enter` becomes the Enter key and `C-c`
becomes an interrupt. Verified against tmux 3.6a, a leading dash is worse —
`send-keys -t %1 -l '-n hello'` fails with `unknown flag -n`, while
`send-keys -t %1 -l -- '-n hello'` succeeds.

So `SendText` uses `send-keys -l -- <text>`, and issues `submit` as a separate
`send-keys <pane> Enter`. The existing layout path is left alone: its input is
layout-file command lines, its behaviour is specified, and changing it is not
this change's business.

### `OpenPane` shells out with a quoted command string

tmux joins the trailing arguments of `new-window` with spaces and hands the
result to a shell, so passing `argv` elements as separate process arguments
loses any element containing a space. The argv is therefore rendered to a single
shell-quoted string.

The quoting logic already exists as `shellQuote` inside `internal/layout`, where
it is a private detail of rendering layout `Command` entries. Rather than
duplicate it or export it from a package about layout, it moves to
`internal/shellquote` — one concept per package, per the project convention —
and `layout` uses it from there. Behaviour is unchanged.

`env` is applied via `new-window -e K=V`, with keys sorted so repeated calls
produce identical command lines (Go map iteration order is otherwise random,
which would make the emitted tmux invocation untestable and needlessly
nondeterministic).

`OpenPane` creates a *window* rather than splitting an existing pane: splitting
requires choosing a target pane and a direction, which is geometry policy this
contract explicitly does not take. The contract says only "a new pane in the
pane group"; window-per-pane is session-tmux's answer and another provider may
answer differently.

### One provider query behind both `ListPanes` and the focus guard

Both read `list-panes -a -F` on the workspace socket with a tab-separated format
of `pane_id`, `session_name`, `pane_title`, `pane_current_command`,
`pane_current_path`, `session_attached`, `window_active`, `pane_active`. A
single parser produces the `Pane` values that `ListPanes` streams and that
`SendText` looks up by ID.

Consequences that fall out for free: `SendText` gets a real `NOT_FOUND` for an
unknown pane without pattern-matching tmux's stderr, and `Pane.focused` cannot
disagree with the guard.

Socket enumeration for `ListPanes({})` reuses the liveness scan `ListWorkspaces`
already performs, extracted into a shared helper so the two cannot diverge on
what counts as a live workspace.

### Error mapping

- Empty required identifier → `INVALID_ARGUMENT`, before any tmux call.
- Unknown workspace socket, pane group, or pane → `NOT_FOUND`.
- Focused pane without `allow_focused` → `FAILED_PRECONDITION`.
- `ClosePane` on a pane that is already gone → success. Cleanup after a program
  that exited on its own is the normal case, not an error. This matches
  `CloseWorkspace`, which is already idempotent, and `killOriginPane`, which
  already swallows "no such pane".

## Risks / Trade-offs

- **The focus guard can be wrong in both directions.** A human away from the
  keyboard with the pane focused gets a refusal; a human who focuses the pane
  microseconds after the check gets the text. Mitigated by documenting it as a
  guard rather than a guarantee, and by exposing `focused` on `Pane` so callers
  can reason about it.
- **Descriptive fields on `Pane` are provider-shaped.** `current_command` is
  tmux's notion of the foreground command. Documented as best-effort and
  zero-valued when unavailable, so a provider that cannot report it stays
  conformant.
- **New methods break out-of-tree Session plugins at compile time**, because
  `require_unimplemented_servers=false`. Accepted: there are no out-of-tree
  session plugins, and the alternative (flipping that option) would silently
  turn missing implementations into runtime `Unimplemented` errors for every
  service in the repo.
- **`shellquote` extraction touches the layout package.** Pure move, covered by
  the existing layout tests plus direct tests on the new package.

## Migration Plan

Additive; no migration. `task proto:gen` regenerates, then
`task update-nix-vendor-hashes` because `proto/` changed. Existing callers
compile and behave identically.

## Open Questions

None blocking. Two deliberately deferred: whether a future `CapturePane` belongs
on this service, and whether `OpenPane` should eventually take a placement hint
once a second provider exists to justify a neutral shape for one.
