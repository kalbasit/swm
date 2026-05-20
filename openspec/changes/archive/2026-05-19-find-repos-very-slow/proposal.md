# Proposal: find-repos-very-slow

## Why

`swm workspace open` takes 4+ seconds because v2 replaced swm-v1's parallel goroutine-per-directory
scan with a serial `filepath.WalkDir`. Additionally, the host has two separate, duplicated scan
implementations (`buildCandidates` in the CLI and `ListProjects` in `hostsvc`) that both walk the
filesystem independently instead of sharing a single result.

## What Changes

- Restore parallel fan-out scanning: enumerate host directories directly (host can never be a repo
  — `ProjectIDFromPath` requires host + at least one segment), then fan out one goroutine per child
  directory starting at the host level; each worker stats for `.git` first and stops if found,
  otherwise fans out further — matching v1's `scanWorker` pattern. Replace both `filepath.WalkDir`
  usages.
- Add an in-process cache to `hostsvc.Server`: scan once per command invocation, memoize the
  `[]*pluginv1.ProjectID` slice, protected by `sync.Once`.
- Remove `buildCandidates`'s own filesystem walk. Instead have it consume the cached result from
  `hostsvc.Server` (passed as a dependency or called via an internal interface), so host-side CLI
  code and plugin-facing gRPC both serve from the same single scan.
- `hostsvc.ListProjects` streams from the cached slice instead of re-walking on every call.

## Non-goals

- On-disk / cross-invocation caching (future work).
- Filesystem-watch invalidation.
- Changing the `repositories/` on-disk layout.
- Reducing plugin-process startup time (separate concern).

## Capabilities

### New Capabilities

_(none)_

### Modified Capabilities

_(none — observable output of `buildCandidates` and `ListProjects` is unchanged; only
implementation strategy changes. No spec-level requirement is being altered.)_

## Impact

- `cmd/swm/internal/hostsvc/server.go` — add `sync.Once` scan cache; `ListProjects` streams from
  cache; expose an internal method/interface for the CLI to consume the cached list.
- `cmd/swm/internal/cli/workspace/open.go` — `buildCandidates` drops its own `WalkDir`; receives
  the repo list from the host server's cache instead.
- `cmd/swm/internal/core/layout/` — new `ScanRepos` function implementing the parallel
  goroutine-per-directory stat+fan-out approach (replaces both walk implementations).
- No proto changes; no new plugins; no config changes.
- Affects: capability surfaces `session` (workspace open) and `host` (ListProjects RPC).
