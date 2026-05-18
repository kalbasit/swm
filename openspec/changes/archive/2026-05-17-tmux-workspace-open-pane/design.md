## Context

`swm workspace open` ends by calling `Session.SwitchTo`, switching the user's terminal to the target workspace. Nothing signals the plugin to close the originating pane, leaving a dangling pane in the source session. v1 addressed this with `swm tmux switch-client --kill-pane`, a tmux-specific CLI call. v2 must express the same behaviour through the plugin protocol so any multiplexer plugin can implement it.

**Key constraint:** the session plugin is a long-lived gRPC daemon. Its process environment is captured at startup and goes stale as the user switches workspaces. The plugin cannot reliably read its own `$TMUX_PANE`; origin information must be passed explicitly in the RPC request.

**Key edge case with `exec_argv`:** when `SwitchTo` returns a non-empty `exec_argv`, the host calls `syscall.Exec` and is replaced. The plugin must kill the origin pane *before* returning the response — after exec the host process no longer exists to act on a kill instruction.

## Goals / Non-Goals

**Goals:**
- Add `--kill-pane` flag to `swm workspace open`
- Extend `SwitchToRequest` with optional origin-pane fields (proto3 backward-compatible addition)
- Implement close-origin in `session-tmux`

**Non-Goals:**
- Zellij plugin implementation (protocol is designed to accommodate it; plugin is out of scope)
- Killing entire source workspaces or sessions (only the single originating pane)
- Changing any other session operation

## Decisions

### Two optional fields on `SwitchToRequest`

Add `close_origin_workspace_id string` (field 3) and `close_origin_pane_id string` (field 4) to `SwitchToRequest`.

**Rationale:** `close_origin bool` alone is insufficient — `session-tmux` uses a socket-per-workspace model and needs the origin workspace to look up the correct socket; it also needs the pane ID to target the kill. A single bool would force the plugin to rely on stale environment variables.

**Alternatives considered:**
- Single `close_origin_ref string` (opaque, plugin-specific): removes structure, makes the host unable to know what to supply without plugin-specific logic.
- gRPC metadata: non-standard for this project's plugin interface; avoided.

Both fields default to `""` in proto3, making the extension fully backward-compatible. No version-number bump is required.

### Host supplies origin via `CurrentContext()` + `$TMUX_PANE`

When `--kill-pane` is set, the host:
1. Calls `Session.CurrentContext()` to obtain the current `workspace_id`.
2. Reads `os.Getenv("TMUX_PANE")` for the multiplexer-specific pane identifier.
3. Passes both in the `SwitchToRequest`. If either is empty (e.g., not inside a multiplexer session), the fields are omitted and `--kill-pane` is silently a no-op.

**Alternative:** parse origin from `$TMUX` socket path. Rejected: socket-to-workspace mapping is plugin-internal state; `CurrentContext()` is the correct API.

### Plugin kills pane before responding when `exec_argv` is involved

`session-tmux` performs the kill inside its `SwitchTo` RPC handler, before returning the response. This ordering is critical: when `exec_argv` is non-empty the host will exec and the RPC channel closes, so any post-response kill would never execute.

Sequence in plugin:
1. Run `tmux switch-client` (or build `exec_argv` for `attach-session`).
2. If `close_origin_pane_id` is set: look up origin socket from workspace registry; run `tmux -S <origin_socket> kill-pane -t <close_origin_pane_id>`; ignore "no such pane" errors.
3. Return `SwitchToResponse`.

### "Not found" kill-pane errors are swallowed

A race (user closes the pane between the switch and the kill, or the origin socket has already shut down) is benign — the pane is gone, which is the desired state. The plugin logs the error at debug level but does not surface it to the host.

## Risks / Trade-offs

- **Race: pane gone before kill** → kill-pane "no such pane" is ignored; end state is correct.
- **Origin workspace not in plugin registry** (e.g., user called `swm workspace open --kill-pane` from a non-swm tmux session) → plugin returns an error; host surfaces it to the user.
- **`CurrentContext()` overhead** → one extra RPC per `workspace open --kill-pane` invocation; negligible.
- **Future multiplexers**: zellij does not use `$TMUX_PANE`; its plugin will read an equivalent env var (e.g., `$ZELLIJ_PANE_ID`) and interpret `close_origin_pane_id` accordingly. The proto field is intentionally untyped.
