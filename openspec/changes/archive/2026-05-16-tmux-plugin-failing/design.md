## Context

Both session-tmux and picker-fzf plugins start, complete the go-plugin handshake, and
report their gRPC addresses. Within ~5 ms, session-tmux exits with a "connection reset
by peer" error (indicating an unclean process exit — panic or `os.Exit`), and
picker-fzf follows with an "EOF" (indicating the host killed it via `client.Kill()`
after the session failure).

**Why the root cause is invisible today**: `goplugin.ClientConfig.SyncStderr` defaults
to `io.Discard`. Any output the plugin binary writes to stderr before or during gRPC
serving — including `fmt.Fprintf(os.Stderr, ...)` followed by `os.Exit(1)` in
`main.go`, or a panic stack trace — is silently discarded. We are diagnosing a crash we
cannot see.

**What we know from code inspection**:
- `session.New()` succeeds (the plugin reaches `Serve()` and reports its address).
- `Info()` likely succeeds for session before picker starts (picker starts 1 ms after
  session's address is reported; `validateDeps` calls `Info()` in that window).
- The crash is asynchronous with respect to any RPC call we can trace — session crashes
  at the exact moment picker's stdio proxy goroutine starts.
- go-plugin prepends `os.Environ()` to `cmd.Env` (SkipHostEnv=false default), so PATH,
  HOME, and XDG_* are all inherited; environment stripping is NOT the root cause.
- No explicit `os.Exit()` or `log.Fatal*()` appears in the plugin code path that could
  be triggered without an error in `New()`.

**Best guess**: The crash is a go-plugin internal keepalive or broker mechanism reacting
to a configuration mismatch, or a panic in an RPC handler triggered by the first real
operation (tmux call). Both require stderr to confirm.

## Goals / Non-Goals

**Goals:**
- Forward plugin stderr to the host log stream so panics, `os.Exit` messages, and error
  traces are visible at DEBUG level.
- Identify and fix the actual root cause of the session-tmux crash (to be confirmed
  after enabling stderr capture).

**Non-Goals:**
- Sandboxing or restricting plugin execution environments.
- Changing the plugin protocol or proto definitions.
- Changing how plugins are discovered or launched (beyond adding stderr forwarding).

## Decisions

### Decision 1: Forward stderr via `SyncStderr` in `ClientConfig`

**Chosen**: Set `SyncStderr: os.Stderr` in every `goplugin.ClientConfig` block inside
`pluginmgr/manager.go` (both `Get()` and `loadForges()`).

**Why `os.Stderr` over an hclog writer**: The simplest change that gives us immediate
visibility. Plugin output will appear on the host's stderr, prefixed by go-plugin with
the plugin path. A more structured approach (hclog writer with plugin-name namespacing)
adds complexity without diagnostic value at this stage.

**Alternative considered**: Use `Logger: hclog.New(...)` per plugin. Rejected because
hclog writer setup requires more code and go-plugin's Logger field controls go-plugin's
own internal log messages, not plugin stderr — we need `SyncStderr` regardless.

### Decision 2: Two-phase fix — stderr first, crash fix second

Run `swm workspace open` again after adding `SyncStderr`. The first task in implementation
is stderr forwarding; the second task is fixing whatever the stderr reveals.

**Why not guess and fix the crash blind**: The crash timing (session dies exactly when
picker's stdio goroutine starts) has no obvious code-level explanation. Guessing could
introduce unrelated changes and still not fix the real issue.

**Hypothesis to validate**: If the crash is from `OpenWorkspace`/`OpenPaneGroup` failing
in a way that panics (e.g., nil pointer dereference from an uninitialized `hostClient`),
the fix is defensive nil-checks. If the crash is a go-plugin broker issue, the fix may
be in gRPC connection or keepalive settings.

## Risks / Trade-offs

- [Risk] Plugin stderr flooding the terminal with verbose output → Mitigation: in a
  follow-up, route `SyncStderr` through a named `slog` or `hclog` writer at DEBUG level
  rather than raw `os.Stderr`.
- [Risk] The crash is in a Nix-built binary that is a different version than the repo
  source → Mitigation: rebuild and install from source after the fix; this does not
  change the investigation approach.
- [Risk] Adding `SyncStderr` changes observable test behaviour → Mitigation: tests use
  injected binaries (faketmux, fakefzf) that do not write to stderr.
