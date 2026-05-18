## Context

`swm workspace open <name>` resolves a story name via positional arg or `$SWM_STORY`, then
calls `store.Get(name)`. When the story does not exist, the error surfaces immediately as a
non-zero exit. There is no hook into the "story not found" path to offer creation.

The story-store `Create` path (called by `swm story create`) is the only way to add a new
story today. Both commands live in the same host binary (`cmd/swm`) and share the same store
dependency, so the create path is accessible from the workspace-open handler without any new
cross-package coupling.

## Goals / Non-Goals

**Goals:**
- When a named story is not found AND stdin is a TTY, prompt `[y/N]` and create the story
  before proceeding with the open flow.
- Preserve exact current behavior for non-TTY stdin (scripts, CI, piped use).
- Reuse the existing story-creation path including `pre-story-create` / `post-story-create`
  hooks.

**Non-Goals:**
- Auto-creation without confirmation.
- Prompting during the picker flow — the picker only lists existing stories; the prompt only
  fires when a name is supplied explicitly via positional arg or `$SWM_STORY`.
- A `--yes` / `--no` flag — the TTY guard is sufficient for scripted safety; adding a flag
  is YAGNI.

## Decisions

### 1 — TTY detection via `golang.org/x/term`

`term.IsTerminal(int(os.Stdin.Fd()))` gates the prompt.

**Alternatives considered:**
- `os.Stdin.Stat()` with `ModeCharDevice` — works but `golang.org/x/term` is already
  vendored (used by the picker path) and its semantics are more explicit.
- Always prompt — breaks non-TTY callers; rejected.

### 2 — Prompt written to stderr, answer read from stdin

The prompt message is written to `os.Stderr`; the answer is read from `os.Stdin` with
`bufio.NewReader`. This keeps stdout clean for any programmatic caller and avoids mixing
the prompt into piped output.

**Alternatives considered:**
- Open `/dev/tty` directly for both prompt and read — more robust to stdout/stderr
  redirection, but unnecessary since the TTY guard already ensures stdin is a terminal.

### 3 — Reuse store.Create + hook invocations from the workspace-open handler

Rather than shelling out to `swm story create`, the workspace-open handler calls
`store.Create(...)` and `hookexec.Run` for `pre-story-create` / `post-story-create`
directly — identical to what `storyCreateCmd` does. This keeps behavior consistent (hooks
fire, duplicate detection works) without process spawning.

**Alternatives considered:**
- Subprocess call to `swm story create` — avoids in-handler code but adds process overhead
  and loses the in-process store handle; rejected.

### 4 — Prompt fires only for explicit names (arg or `$SWM_STORY`)

Story resolution order is: positional arg → `$SWM_STORY` → picker → `_default`. The prompt
fires only at steps 1 and 2, because those are the only steps where a user-supplied name
could be non-existent. The picker enumerates only existing stories (step 3), and `_default`
always exists (step 4).

## Risks / Trade-offs

- [Risk] A `pre-story-create` hook aborts creation — the workspace open also aborts with a
  non-zero exit. → Matches `swm story create` behavior; expected.
- [Risk] Stdin read could interfere with a subsequent picker if both read from stdin. →
  Not possible: the prompt fires only when the name is explicit (steps 1/2); the picker is
  invoked only when no explicit name is given (steps 3/4). The paths are mutually exclusive.

## Migration Plan

No migration needed. The change is purely additive: non-TTY callers see no behavioral change.
Interactive users gain the prompt; existing muscle memory (`swm story create` then
`swm workspace open`) still works unchanged.

## Open Questions

_(none)_
