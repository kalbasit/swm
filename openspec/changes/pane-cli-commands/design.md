## Context

`session-pane-primitives` deliberately shipped with "No CLI surface" as a
non-goal, so the four pane RPCs exist but nothing outside the host can reach
them. This change adds only the seam.

The consumer that motivates it is a program, not a person: a per-host agent that
opens a pane, remembers the handle, sends text into it, and closes it later.
That single fact drives every decision below — output has to be parseable,
identifiers have to round-trip verbatim, and the one refusal a caller must
handle programmatically has to be detectable without reading English.

## Goals / Non-Goals

**Goals**

- One CLI subcommand per RPC, adding no behaviour of its own.
- Machine-readable output for the two commands that produce output.
- A programmatic signal for the focused-pane refusal.

**Non-Goals**

- Inferring the workspace from the ambient session.
- Any convenience that re-derives a `pane_id` or interprets it.
- Human-facing polish beyond a readable default table.

## Decisions

### `swm pane <verb>`, matching the existing group shape

`story`, `workspace`, `pr` and `config` are all groups of subcommands, so a
`pane` group with `open` / `list` / `send` / `close` is the shape the repo
already uses. The verbs are the RPC names lowercased, minus the `Pane` suffix
that the group already supplies: `OpenPane` → `pane open`. `SendText` becomes
`pane send` rather than `pane send-text`, because the group noun already says
what is being sent to and the object of the verb is the command's argument.

The group is built by `pane.NewPaneCmd(mgr)` rather than assembled inline in
`root.go` like `story` and `workspace`, following `config.NewConfigCmd`: four
subcommands that all need the same session client is enough to be worth a
constructor of its own.

### `--workspace` is required, never inferred

`Pane.pane_id` is documented as meaningful only within the workspace that
produced it. A CLI that guessed the workspace would make the handle ambiguous
exactly where the contract says it must not be. So `open`, `send` and `close`
all take `--workspace` and it has no default.

`list` is the exception in the other direction: `ListPanesRequest` documents an
empty `workspace_id` as "every live workspace", and that is precisely the call a
caller makes first, before it has any workspace ID to pass. So on `list` both
filters are optional and default to unset, which is how a caller discovers
workspace IDs in the first place.

There is a real cost here: a caller's first run is always `swm pane list --json`
to learn the workspace ID. That is accepted rather than papered over, because
the alternative — inferring from `$SWM_STORY` or `CurrentContext` — silently
picks a workspace when the caller is wrong about which one it is in, and the
failure would be a pane opened in the wrong place.

### JSON is a hand-written struct, not protojson

`protojson` would emit `paneId`, `paneGroupId`, `currentCommand`. The proto
field names are snake_case and every other machine-facing name in this project
is snake_case, so the CLI marshals through a small struct with explicit
`json` tags instead. Every field is always present, including empty
descriptive ones, so a consumer can index without existence checks — the
best-effort fields are documented as zero-valued when the provider cannot
supply them, and an absent key would express that less clearly than a present
empty one.

`list --json` emits a JSON array, not newline-delimited JSON, and emits `[]`
rather than `null` for no panes. An array is what `jq` consumes without flags;
pane counts on a host are small enough that buffering the stream costs nothing.

### `open` prints the bare `pane_id` by default

The default (non-`--json`) output of `open` is the pane ID and a newline,
nothing else — the whole point is `pane=$(swm pane open ...)`. Any extra
decoration would have to be stripped by every caller. `--json` gives the full
`Pane` for a caller that also wants the pane group or path back.

### The focused refusal gets its own exit status

A caller has to be able to branch on "refused because a human is typing there"
without matching on message text, because that is the one failure with a
different correct response: back off and retry, rather than fail. The plugin
contract already says this failure is `FAILED_PRECONDITION` from `SendText`,
so the CLI maps that one status code to exit 3 and leaves every other failure
on exit 1.

`FAILED_PRECONDITION` from `SendText` is treated as the focused refusal
wholesale, rather than trying to recognise the message. The contract assigns
that code to exactly this condition on this RPC, so a provider using it for
something else is out of conformance; matching on text would be worse in every
way.

3 rather than 2: 2 is conventionally a usage error in POSIX-ish CLIs, and
reserving it costs nothing here.

Carrying the code out of the command needs a channel, because production code
outside `main` must not call `os.Exit`. `cmd/swm/internal/exitcode` gives
errors an optional `ExitCode() int`, and `main` asks the returned error for one
via `errors.As`, defaulting to 1. That keeps process termination in `main`,
where the project rules require it, and leaves the mechanism available to any
future command without another bespoke path.

### `send` validates the empty delivery locally

`SendTextRequest` documents empty text without `submit` as `INVALID_ARGUMENT`.
The CLI rejects it before dialling the plugin, because the resulting message can
name the actual flags (`--submit`) instead of the proto fields, and because
starting a plugin process to be told the arguments were wrong is wasted work.
Every other validation stays where the contract puts it: on the plugin.

### `argv` is positional, `env` is repeatable `K=V`

`swm pane open -w W -g G -- nvim file.txt` passes `[nvim file.txt]` as `argv`
verbatim; cobra hands everything after `--` through untouched, which is exactly
the "already split, never re-split" rule the request field documents. No argv is
an empty `argv`, which the contract defines as the provider's default shell.

`--env K=V` may be repeated. A value may contain `=`; only the first one splits.
An entry with no `=` or an empty key is a local error rather than something the
plugin has to guess at.

## Risks / Trade-offs

- **Exit code 3 is a new contract of its own.** Once a caller branches on it, it
  cannot be renumbered. Accepted: it is specified, and the alternative (parse
  stderr) is worse and unversionable.
- **A non-conforming provider could return `FAILED_PRECONDITION` from `SendText`
  for something else**, and the CLI would report it as a focused refusal.
  Accepted; the contract is explicit about that code's meaning on that RPC.
- **Requiring `--workspace` makes the first call a discovery call.** Accepted
  above; inference is the more dangerous default.

## Migration Plan

Purely additive. No existing command, flag, or output changes. No proto change,
so no regeneration and no Nix vendor hash update.

## Open Questions

None blocking. Deferred: whether `--workspace` should eventually accept a story
name as well as a workspace ID, which is a general question about every command
that takes one and not specific to panes.
