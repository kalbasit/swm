## Why

The `session-pane-primitives` change put `OpenPane`, `ListPanes`, `SendText`
and `ClosePane` on the `Session` service, but they are reachable only by a
process that can speak the plugin gRPC contract — in practice, only `swm`
itself.

The callers these primitives were built for are not plugins. A per-host agent
that wants to start a program in a pane and drive it is a plain external
process. Today it has exactly two options: implement the go-plugin handshake,
or shell out to `tmux` directly — and shelling out to `tmux` is the thing the
whole capability model exists to prevent, because it does not survive a second
provider. The CLI is the seam that makes the primitives usable without either.

## What Changes

- Add a `swm pane` command group with four subcommands — `open`, `list`,
  `send`, `close` — each a thin wrapper over the matching `Session` RPC. No new
  behaviour is introduced; nothing is interpreted or re-derived on the way
  through.
- `open` and `list` gain `--json` for machine consumption. `open` prints the new
  `pane_id` on stdout in its default mode so a caller can capture it with a
  bare command substitution.
- `send` maps the plugin's focused-pane refusal to a dedicated exit status (3),
  distinguishable from any other failure without parsing stderr, and names
  `--allow-focused` in the error text.
- Add `cmd/swm/internal/exitcode` so a command can request a specific process
  exit status without calling `os.Exit` itself, and teach `main` to honour it.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `workflow-commands`: gains the `swm pane` command group and the exit-status
  contract that `swm pane send` relies on.

## Non-goals

- No new plugin surface. No proto change, no RPC change, no session-tmux
  change. This change is CLI-only; the wire contract it exposes is already
  specified by `session-pane-primitives`.
- No workspace inference. `--workspace` is required wherever a `pane_id` is,
  because a `pane_id` means nothing outside the workspace that minted it.
  Resolving it from the ambient session is a separate question.
- No new vocabulary. The commands say pane, exactly as the contract does; swm
  still does not model what runs in one.
- No output capture, no waiting for a program to finish, no supervision.
- No relaxation of the focused-pane guard. The CLI surfaces the refusal; it
  does not decide to bypass it, and never sets `allow_focused` on its own.

## Impact

- `cmd/swm/internal/cli/pane` — new package (four commands plus JSON encoding).
- `cmd/swm/internal/exitcode` — new package.
- `cmd/swm/internal/cli/root.go`, `cmd/swm/main.go` — wiring.
- `openspec/specs/workflow-commands/spec.md` — four new command requirements
  plus the exit-status requirement.
- No `proto/`, `sdk/go`, or `plugins/` change, so no `task proto:gen` and no
  `task update-nix-vendor-hashes`.
