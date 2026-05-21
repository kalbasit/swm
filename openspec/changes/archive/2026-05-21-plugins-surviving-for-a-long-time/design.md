## Context

`swm workspace open` discovers, warms, and uses plugins (picker, session, vcs)
to drive the workspace selection flow. At the end of a successful run it calls
`syscall.Exec` to replace the swm process with `tmux attach-session`. Because
`exec` replaces the process image, all deferred calls — including the
`defer mgr.Close()` in `main.go` — are skipped. Plugin subprocesses inherit
the exec'd process's PID ancestry and become children of the tmux client;
when the user detaches, the client exits and they are reparented to init as
orphans.

## Goals / Non-Goals

**Goals:**
- Ensure no plugin subprocess outlives a `swm workspace open` invocation.
- Minimal change: touch only the `pluginManager` interface and the exec call site.

**Non-Goals:**
- Reusing existing plugin processes across invocations.
- Tying plugin lifetime to the tmux session or tmux server.
- Changes to plugin discovery, launch, or warm logic.

## Decisions

### Add `Close() error` to the `pluginManager` interface

The `pluginManager` interface in `cmd/swm/internal/cli/workspace/open.go`
currently exposes only `Get` and `Warm`. `*pluginmgr.Manager` already
implements `Close() error`; it just isn't part of the interface the open
command uses.

**Why not pass a separate `closePlugins func()`?** An extra function parameter
adds surface area for tests and callers. Extending the existing interface keeps
the dependency injection pattern uniform and requires zero new wiring in
`main.go`.

**Alternatives considered:**
- Call `mgr.Close()` inside `execFn` wrapper — leaks lifecycle concerns into
  a function whose only job is process replacement; makes testing harder.
- Remove the `defer mgr.Close()` in `main.go` and rely solely on the explicit
  call — the defer is a useful safety net for error paths that return before
  exec; keeping it is cheap and correct (double-close is a no-op in go-plugin).

### Call `mgr.Close()` immediately before `execFn()`

The call goes just before the `execFn` invocation at the end of the open flow,
after the post-workspace-open hook has run. A `Close()` error is logged but
does not block the exec — the user's workflow should not be interrupted because
a plugin cleanup returned an error.

**Why not return the error?** After `SwitchTo` has already switched the tmux
client, there is no useful recovery action. The workspace is open; the exec
is the only remaining step. Surfacing the error would leave the user with a
confusing failure message after their pane group is already visible.

## Risks / Trade-offs

- **Double-close on error paths**: If `workspace open` returns an error before
  exec, the deferred `mgr.Close()` in `main.go` closes the manager. If a future
  code path calls `mgr.Close()` explicitly and then hits the defer, the second
  close is a no-op per go-plugin's `Kill()` contract. No risk.
- **Log noise on Close error**: Logging a Close error just before exec could
  appear after the user's terminal is handed to tmux. Acceptable: it's
  unexpected and worth surfacing.

## Open Questions

None. The change is small and the decision space is exhausted.
