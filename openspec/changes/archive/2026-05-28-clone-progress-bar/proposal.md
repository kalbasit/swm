## Why

`swm clone` silently waits while cloning large repositories — there is no progress feedback, leaving users uncertain whether the operation is running or stalled. Since `git clone` already emits progress to stderr, we should surface it through the plugin protocol and render it in the terminal.

## What Changes

- **BREAKING**: Replace the unary `Clone(CloneRequest) returns (CloneResponse)` RPC in `proto/swm/plugin/v1/vcs.proto` with a server-streaming `Clone(CloneRequest) returns (stream CloneProgressEvent)`. `CloneResponse` is removed; `CloneProgressEvent` carries both progress lines and the final `ProjectID`.
- Update `swm-plugin-vcs-git` to implement the new streaming `Clone`: pipe git's stderr live through the gRPC stream and send a terminal event with the resolved `ProjectID` on success.
- Update `cmd/swm clone` to consume the stream and render a live progress indicator to the terminal (forwarding git's progress lines verbatim or using a simple spinner/bar).

## Non-goals

- Parsing or structuring git's progress output (percentages, object counts) — forwarding raw lines is sufficient.
- Adding progress to `AddWorktree` or other VCS operations.
- Supporting non-git VCS plugins in this change.

## Capabilities

### New Capabilities

_(none)_

### Modified Capabilities

- `vcs-git` — replaces unary `Clone` with streaming `Clone`; requirement changes: clone must emit progress events to the caller instead of running silently, and return the resolved `ProjectID` as a terminal stream event.

## Impact

- `proto/swm/plugin/v1/vcs.proto` — replace unary `Clone` RPC with streaming variant; remove `CloneResponse`; add `CloneProgressEvent`; regenerate `.pb.go` files; run `task update-nix-vendor-hashes` after.
- `plugins/vcs-git/internal/vcs/git.go` — implement streaming `Clone`; pipe git stderr live; send terminal event with `ProjectID`.
- `cmd/swm/internal/cli/clone.go` — consume the stream; render progress to stderr.
- Affected capability surface: **vcs**.
- Proto note: breaking change within `proto/swm/plugin/v1/` — all VCS plugin implementations must be updated.
