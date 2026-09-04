## Why

swm owns all multiplexer interaction: any other program that wants to drive tmux
must go through the `session` capability, because a second provider (zellij,
wezterm) may be introduced and direct `tmux` calls would not survive it.

But the `Session` service has no pane-level vocabulary. Its nouns are Workspace
and PaneGroup; there is no way to enumerate panes, start a program in a new pane,
deliver text to one, or close one. Text delivery exists in exactly one place —
`plugins/session-tmux/internal/layout/apply.go`, driven by a layout file at
pane-group-creation time — and is not reachable at runtime. Consequently a caller
cannot even answer "what is running on this host right now" through swm, and any
program needing to start and drive a process in a pane has no option but to shell
out to `tmux` directly, which is exactly what the ownership rule forbids.

## What Changes

- Add four RPCs to `Session`: `OpenPane`, `ListPanes` (server-streaming),
  `SendText`, `ClosePane`, plus the `Pane` message and the four request
  messages. Additive only — no existing RPC, message, or field number changes.
- Implement all four in `plugins/session-tmux`.
- `pane_id` is an opaque provider-specific handle (`%4` on tmux). Callers pass it
  back verbatim and never parse it. This is not a new kind of value: the service
  already carries one in `SwitchToRequest.close_origin_pane_id`, documented there
  as "the multiplexer-specific pane reference (e.g. `$TMUX_PANE` for
  session-tmux)".
- `SendText` refuses by default to deliver into a pane the multiplexer reports as
  focused by an attached client, because injected text can be consumed by a
  prompt a human is answering. `allow_focused` is the explicit opt-out.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `plugin-protocol`: the `Session` wire contract gains pane-level primitives and
  the opaque-pane-handle rule.
- `session-tmux`: gains the tmux implementation of those four RPCs.

## Non-goals

- No CLI surface. No `swm` command is added; this change is the plugin contract
  and its tmux implementation only. `openspec/specs/workflow-commands/spec.md` is
  untouched.
- No knowledge of what runs in a pane. These are generic multiplexer primitives;
  swm does not model AI agents, jobs, or supervision, and gains no vocabulary for
  them.
- No pane geometry control (splitting, resizing, focusing). Where a provider puts
  a new pane is provider policy; layout remains the layout config's job.
- No output capture (`capture-pane`) and no attach/detach semantics.

## Impact

- `proto/swm/plugin/v1/session.proto` — additive; requires `task proto:gen` and
  `task update-nix-vendor-hashes`.
- `plugins/session-tmux` — new handlers, a shell-quoting helper package, and
  faketmux support for `new-window` / `list-panes`.
- `cmd/swm` — unchanged. `sdk/go` — unchanged (`session.Plugin` is a type alias
  for `pluginv1.SessionServer`, so the new methods flow through automatically).
- Any out-of-tree `Session` plugin must implement the four new methods;
  `require_unimplemented_servers=false` means this is a compile-time break for
  such plugins. There are none in this repo besides session-tmux.
