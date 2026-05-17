## Context

`swm workspace open` calls `session.SwitchTo` after opening a pane group. The
session-tmux plugin's `SwitchTo` handler runs `tmux attach-session` when the
user is not already inside a tmux session. This fails with "not a terminal"
because the plugin is a gRPC subprocess — its stdin/stdout are pipes managed
by go-plugin, not the user's terminal. The terminal is held by the swm host
process.

The fix must allow the plugin to convey "run this command with my terminal"
back to the host, which then uses `syscall.Exec` to replace itself with the
tmux process and inherit the terminal correctly.

## Goals / Non-Goals

**Goals:**
- `swm workspace open` attaches the user to the tmux session when not already
  inside tmux
- The plugin controls the exact attach command (socket path, session name)
- The host provides the terminal by exec-replacing itself

**Non-Goals:**
- Changing `switch-client` behaviour (already works when inside tmux)
- Supporting non-tmux session backends differently (they leave `exec_argv`
  empty)
- Handling deferred attach (e.g. daemonised attach)

## Decisions

### 1. Extend `SwitchToResponse` with `repeated string exec_argv`

**Decision:** Add `SwitchToResponse { repeated string exec_argv = 1; }` to
`session.proto` and change `rpc SwitchTo` to return it instead of `Empty`.

**Alternatives considered:**

- *Return `codes.FailedPrecondition` from `SwitchTo`* — lets the host detect
  "please attach", but the host then has to reconstruct the tmux socket path
  and session name from the workspace/pane IDs it already holds. Fragile: it
  couples the host to tmux internals and breaks if socket naming changes.

- *New `rpc AttachCommand(…) returns (AttachCommandResponse)`* — clean
  separation but adds a round-trip, and the attach command is logically part
  of `SwitchTo`'s contract ("how to bring this into focus").

- *Out-of-band file / env var* — avoids proto change but is a global side
  channel and untestable.

`exec_argv` keeps the protocol self-contained: the plugin returns exactly what
the host must exec, and the host stays agnostic to tmux specifics.

### 2. `syscall.Exec` on the host side (not `exec.Command`)

**Decision:** When `exec_argv` is non-empty, the host calls
`syscall.Exec(exec_argv[0], exec_argv, os.Environ())`. This replaces the host
process in-place, inheriting its file descriptors (including the terminal).

**Why not `exec.Command`?** Running tmux as a child process has the same TTY
problem: the child inherits the host's pipes from go-plugin, not the
terminal. `syscall.Exec` is the only way to pass control of the terminal to
tmux.

**Side effect:** `syscall.Exec` never returns on success, so the
post-workspace-open hook cannot run after attach. The hook is run *before* the
exec call to preserve hook semantics.

### 3. Plugin returns `exec_argv` only when `$TMUX == ""`

**Decision:** The plugin checks `os.Getenv("TMUX")` at the start of
`SwitchTo`. When empty (not inside tmux), it returns
`exec_argv = [tmuxBin, "-S", sock, "attach-session", "-t", target]` without
running tmux. When non-empty, it calls `switch-client` (existing behaviour)
and returns empty `exec_argv`.

This keeps the exec path entirely in the plugin, preserving the capability
abstraction.

### 4. Proto version bump

The proto package remains `swm.plugin.v1`. The `SwitchToResponse` message is
new and the existing `Empty` return is replaced, so all generated code
(`session.pb.go`, `session_grpc.pb.go`) must be regenerated via
`task proto:gen`.

## Risks / Trade-offs

- **`syscall.Exec` is non-returning** → post-workspace-open hook must run
  before exec; this changes hook timing slightly but is semantically correct
  (the hook fires when the workspace is "open", before handing off the
  terminal).
- **Proto breaking change** → any external plugin implementing the `Session`
  service must be updated. In practice only `session-tmux` exists today.
- **`repeated string exec_argv` is unchecked** → the host execs whatever the
  plugin returns; a malicious/buggy plugin could exec arbitrary commands. This
  is accepted: plugins are trusted executables already running as the same
  user.

## Open Questions

_(none — all decisions resolved)_
