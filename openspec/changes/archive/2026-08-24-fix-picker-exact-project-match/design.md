## Context

See `proposal.md — Why` for the motivation and the reproduction.

The constraint that shapes this design is tmux's target grammar. `tmux(1)`, under
COMMANDS, specifies that a `target-session` is resolved by trying, in order:

1. exact session name,
2. session name **prefix**,
3. `fnmatch(3)` pattern.

Verified empirically against tmux 3.6a: with a session `abc-name-xyz`, the targets `abc`
(prefix), `abc*` and `*name*` (fnmatch) all resolve to it, while the bare substring `name`
does **not**. Substring matching is not part of `target-session` resolution.

and that "if the session name is prefixed with an `=`, only an exact match is accepted".
The same `=` escape applies to `target-window` and `target-pane` when those are given by name.

`session-tmux` derives pane-group names from project IDs (`sessionName()` maps
`host/segments...` to a tmux-safe name, substituting `.` → `•` and `:` → `：`). That substitution
sanitises characters but does nothing for ambiguity: `git•entreprise•com/name` is still a
strict prefix of `git•entreprise•com/name-two`, so rule 2 fires.

Two other properties matter for the design:

- The plugin already routes every tmux invocation through a single `run(ctx, args...)` helper
  in `session/tmux.go` and a package-level `run` in `layout/apply.go`. There is one place per
  package where argv is assembled.
- The `faketmux` test double models `has-session` as a flat env-var toggle
  (`FAKETMUX_HAS_SESSION`), with no notion of which sessions exist. That is exactly why the
  existing suite is green despite the bug — the fake cannot express "a *different* session
  exists".

## Goals / Non-Goals

**Goals:**

- Every tmux target given as a *name* is resolved exactly.
- A regression test that fails against today's code and passes after the fix.
- The test double models enough of tmux's resolution order that this class of bug is
  catchable, not just this instance of it.

**Non-Goals:**

- Changing how pane-group or workspace names are generated. Names on disk stay byte-identical,
  so pane groups created by older versions stay reachable.
- Making `sessionName()` collision-proof in general. The `.` → `•` substitution can still map
  two distinct project IDs onto one tmux name (`a.b/c` and `a•b/c`); that is a separate,
  far more obscure defect and is out of scope here.
- Auditing tmux invocations outside `session-tmux`. There are none — the host never shells out
  to tmux.

## Decisions

### D1: Escape at the call sites, not inside `run()`

Prefix each name target with `=` where the argv is assembled, rather than teaching the `run`
helpers to rewrite any argument following `-t`.

*Why:* only the call site knows whether the value after `-t` is a **name** (needs `=`) or a
tmux-assigned **ID** (`%1`, `@2`, `$3` — must not be escaped; `=%1` is not valid target syntax).
A blanket rewrite inside `run()` would corrupt the pane-ID targets used by `select-pane`,
`resize-pane`, `send-keys`, and `kill-pane`.

*Alternative considered:* a `target(name string) string` helper returning `"=" + name`, applied
at call sites. This is D1 plus a named function; worth using for readability, and it gives one
place to attach the explanatory comment. Adopted as the concrete form of D1.

*Alternative rejected:* resolving names to session IDs (`$N`) via `list-sessions -F` and
targeting by ID. Correct, but it adds a round-trip to every operation and a failure mode
(session killed between resolve and use) to fix a problem that a one-character prefix solves.

### D2: Which call sites change

| File | Command | Target kind | Change |
|---|---|---|---|
| `session/tmux.go:226` | `has-session` | name | `=` |
| `session/tmux.go:304` | `switch-client` | name | `=` |
| `session/tmux.go:311` | `attach-session` | name | `=` |
| `session/tmux.go:358` | `kill-pane` | pane ID | none |
| `layout/apply.go:28` | `setenv` | name | `=` |
| `layout/apply.go:52` | `rename-window` | name + `:0` | `=` |
| `layout/apply.go:57` | `new-window` | name | `=` |
| `layout/apply.go:86,94,201` | `select-pane`, `resize-pane`, `send-keys` | pane ID | none |
| `layout/apply.go:209` | `display-message` | name or pane ID | `=` only for the name form |
| `layout/apply.go:227` | pane-ID arg | pane ID | none |

`new-session -s <name>` and `new-session -d -s <name>` are **not** targets — `-s` names the
session being created. Unchanged.

The `layout/apply.go` sites operate on a session `OpenPaneGroup` just created, so they are far
less likely to misfire in practice. They are fixed in the same pass because the argument for
leaving a known-ambiguous target unescaped is nil, and a half-escaped file invites the next
reader to copy the wrong pattern.

Note `rename-window -t <name>:0`: the `=` binds to the session component, giving `=<name>:0`.

### D3: Teach `faketmux` to track sessions and mimic tmux resolution

Replace the `FAKETMUX_HAS_SESSION` toggle with real state: `new-session` appends its `-s` name
to a sessions file next to the socket; `has-session` reads that file and resolves its `-t`
argument using tmux's own order — exact, then prefix, then fnmatch — **honouring a leading `=`
as exact-only**.

*Why:* a fake that answers "does session X exist?" from an env var cannot represent the state
that triggers this bug (a *different* session exists). Without this, the regression test can
only assert on recorded argv — which proves we passed `=` but not that `=` produces the right
behaviour. Modelling the resolution order gives a test that would also catch a future
regression that drops the escape somewhere else.

`FAKETMUX_HAS_SESSION` is used by existing tests; keep it as an override that short-circuits
the lookup so those tests need no changes, and let the stateful path run when it is unset.

*Alternative rejected:* an integration test against real tmux. `cmd/swm/tests/integration`
already runs real plugin binaries, and a real-tmux test would be the most faithful check — but
it needs a tmux binary and a usable TTY-less server on every CI runner, which is a much larger
lift than teaching the existing fake three lines of matching logic. If a real-tmux integration
test is added later it complements, not replaces, this.

### D4: Assert on behaviour, not just argv

The regression test drives `OpenPaneGroup` for `<host>/name-two`, then `OpenPaneGroup` for
`<host>/name`, and asserts that a second `new-session` was issued and that the two pane groups
have distinct IDs. The existing argv assertions in `tmux_test.go` (lines ~232, ~695) are updated
to expect `=<name>`, which keeps a cheap direct check on the escape itself.

## Risks / Trade-offs

- **A `=` is added where the target is actually an ID** → the table in D2 enumerates every site
  and its target kind; pane IDs are left alone by construction. The `layout` pane-ID path flows
  through `targetArgs`/`r.paneID`, which is textually distinct from the name path.
- **`=` support in old tmux versions** → the `=` exact-match prefix predates tmux 1.9 and is not
  version-gated. No minimum-version bump.
- **`faketmux` drifts from real tmux** → the fake models the resolution order, not tmux. It
  will not catch a bug that depends on tmux behaviour outside that order. Accepted: the fake's
  job is to make the ambiguity expressible, and a comment in the fake should say it deliberately
  mirrors the tmux(1) target-resolution rules so a future reader knows why it is there.
- **Existing tests coupled to `FAKETMUX_HAS_SESSION`** → kept working as an explicit override
  rather than deleted, so this change does not turn into a test-suite rewrite.

## Migration Plan

None required. Session and window names are unchanged; only the addressing is. A user with a
running workspace can upgrade the plugin in place — the next `swm workspace open` addresses the
same names exactly and finds the same sessions. Rollback is a plain revert with the same
property in reverse.
