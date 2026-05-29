## Context

`swm clone` calls the VCS plugin's `Clone` RPC as a unary call. The plugin runs `git clone` via `exec.Cmd.Output()`, which buffers all output until the process exits. No feedback reaches the user during the clone. For large repositories this looks like a hung process.

`git clone` already emits structured progress to stderr (object counts, percentages, transfer speeds) using `\r`-terminated lines when `--progress` is passed. The fix is to make the RPC server-streaming so the plugin can push those lines to the host in real time.

## Goals / Non-Goals

**Goals:**
- Replace the unary `Clone` RPC with a server-streaming variant that emits progress lines during the clone.
- Render git's native progress output in the terminal as the clone runs.
- Deliver the resolved `ProjectID` via the stream (eliminating the need for a separate `CloneResponse`).

**Non-Goals:**
- Parsing or structuring git's progress output (percentages, object counts) — forwarding raw lines is sufficient.
- Adding a progress bar library; forwarding git's `\r`-based lines to stderr is enough.
- Adding progress to `AddWorktree` or other VCS operations.
- Backwards compatibility shims — there are no external plugin implementations yet.

## Decisions

### 1. `CloneProgressEvent` uses a `oneof` to multiplex progress and result

```protobuf
message CloneProgressEvent {
  oneof event {
    string progress_line = 1;  // raw line from git stderr
    ProjectID project_id  = 2; // terminal event: clone succeeded
  }
}
```

**Why**: A single stream carries both in-flight progress and the final result without needing a second RPC or an out-of-band return value. The host reads until EOF; the last event is always a `project_id`.

Alternative considered: keep `CloneResponse` as a trailer metadata field. Rejected — gRPC trailers in Go require more ceremony and the `oneof` pattern is already established by `ListBranches` returning `stream Branch`.

### 2. Pass `--progress` to git and pipe stderr line-by-line

The plugin switches from `cmd.Output()` to `cmd.StderrPipe()` + a goroutine that reads and streams each `\r`- or `\n`-terminated segment. `--progress` forces git to emit progress even when stderr is not a TTY (which it won't be — it's a pipe to the gRPC stream).

**Why**: Minimal change to the plugin's structure; `StderrPipe` is the standard Go way to read subprocess stderr incrementally.

### 3. CLI forwards raw progress bytes to its own stderr

The host reads each `CloneProgressEvent` from the stream. For `progress_line` events it writes the line directly to `cmd.ErrOrStderr()`. For the terminal `project_id` event it stores the ID and continues to drain the stream until EOF, then uses the ID for the post-clone hook and final message.

**Why**: Git's `\r`-based in-place updates render correctly when written to a real TTY. No extra library needed. If stdout/stderr is not a TTY (e.g., piped to a script) the progress lines are still visible on stderr, matching `git clone` behaviour.

### 4. `CloneResponse` proto message is removed

It only contained `project_id`, which is now the terminal `CloneProgressEvent`. Removing it keeps the proto surface clean.

## Risks / Trade-offs

- [git stderr encoding] git writes progress using `\r` without `\n`; the line-reader must split on both `\r` and `\n` to avoid buffering an entire in-place update sequence as one giant line. → Mitigation: use a `bufio.Scanner` with a custom split function that treats `\r` and `\n` as delimiters.
- [stream back-pressure] If the host is slow to read (e.g., a future non-interactive consumer), the gRPC send buffer fills and the plugin blocks. → Acceptable: progress lines are small; real-world clones won't saturate the buffer.
- [proto regeneration] Changing the RPC signature regenerates `.pb.go` and `_grpc.pb.go` in `proto/` and invalidates all five Nix `vendorHash` values. → Mitigation: run `task update-nix-vendor-hashes` as part of the implementation.

## Migration Plan

1. Edit `proto/swm/plugin/v1/vcs.proto`: remove `CloneResponse`, add `CloneProgressEvent`, change `Clone` signature.
2. Run `task proto` (or equivalent) to regenerate `.pb.go` files.
3. Update `plugins/vcs-git` to implement the new `Clone` server-streaming method.
4. Update `cmd/swm/internal/cli/clone.go` to consume the stream.
5. Run `task update-nix-vendor-hashes`.
6. Run `task fmt lint test` to verify.

No rollback complexity — this is a monorepo change with no external consumers.
