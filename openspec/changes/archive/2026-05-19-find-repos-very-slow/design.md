# Design: find-repos-very-slow

## Context

`swm workspace open` (and any command that shows a project picker) takes 4+ seconds.
The bottleneck is two independent, serial `filepath.WalkDir` calls that visit every
filesystem entry under `$CODE_ROOT/repositories/`:

- `buildCandidates` in `cmd/swm/internal/cli/workspace/open.go` — host-side, feeds the picker.
- `hostsvc.Server.ListProjects` in `cmd/swm/internal/hostsvc/server.go` — gRPC endpoint that
  plugins can call.

swm-v1 solved this with a goroutine-per-directory fan-out (via `sync.WaitGroup` + channel).
v2 regressed to serial walking when the plugin architecture was introduced.

Currently no production plugin calls `ListProjects`, but the duplicate implementations mean
any future caller would trigger a second full scan.

## Goals / Non-Goals

**Goals**
- Restore sub-second repo discovery for typical code roots (100–500 repos).
- Eliminate the duplicated scan: one scan per invocation, shared by all consumers.
- Keep the existing observable output identical (same repo set, same `ProjectID` shape).

**Non-Goals**
- On-disk / cross-invocation caching.
- Filesystem-watch invalidation.
- Bounded goroutine pool (repo trees are shallow; unbounded fan-out is safe in practice).
- Plugin-process startup latency (separate concern).

## Decisions

### 1. Extract `(*Resolver).ScanRepos` into the `layout` package

**Decision**: Add `ScanRepos(ctx context.Context) ([]*pluginv1.ProjectID, error)` to
`layout.Resolver`. It owns the parallel scan logic. Both `hostsvc` and the workspace CLI
call it through this single entry point.

**Alternatives considered**:
- Inline in `hostsvc`: keeps scan hidden from layout but re-couples it to gRPC types.
- Separate `reposcanner` package: unnecessary indirection for a single function.

**Rationale**: `Resolver` already owns the mapping between filesystem paths and
`ProjectID`. Scan logic is a natural extension of that responsibility. One package,
one concept.

---

### 2. Skip `.git` stat at the host level

**Decision**: `ScanRepos` calls `os.ReadDir` on `repositories/` and treats every
entry as a host directory without checking for `.git`. Goroutine fan-out begins at the
host's children.

**Rationale**: `ProjectIDFromPath` requires `len(parts) >= 2` (host + at least one
segment), so a host directory itself can never be a repo. Statting `.git` there is
always wasted work. This is an invariant of the on-disk layout, not a heuristic.

---

### 3. Parallel goroutine-per-directory fan-out (v1 pattern)

**Decision**: Each worker for a directory:
1. Stats `<dir>/.git`. If found → emit `ProjectID`, return (do not descend).
2. If not found → `os.ReadDir` the directory, spawn one goroutine per child dir.

Use `golang.org/x/sync/errgroup` with the incoming context for structured concurrency
and clean error propagation.

**Alternatives considered**:
- Bounded worker pool (e.g. `semaphore`): adds complexity; repo trees are 2–4 levels
  deep so goroutine count is bounded naturally by the tree structure.
- `filepath.WalkDir` with `SkipDir` optimization: serial; demonstrated to be too slow.

---

### 4. Cache in `hostsvc.Server` via `sync.Once`

**Decision**: Add a `sync.Once`-guarded `Projects(ctx) ([]*pluginv1.ProjectID, error)`
method to `hostsvc.Server`. Both `ListProjects` (gRPC) and the workspace CLI call
`Server.Projects()`. The scan runs at most once per process lifetime.

**Alternatives considered**:
- Cache in `layout.Resolver`: Resolver is a stateless value type; adding a cache there
  would require making it a pointer and adding a mutex for little benefit.
- Scan eagerly at server start: harder to plumb context; `sync.Once` is lazy and simpler.

**Limitation**: `sync.Once` does not retry on context cancellation. In practice
`workspace open` uses a single long-lived context, so this is acceptable.

---

### 5. `ProjectLister` interface defined in the `workspace` package

**Decision**: Define:

```go
// In cmd/swm/internal/cli/workspace/open.go
type projectLister interface {
    Projects(ctx context.Context) ([]*pluginv1.ProjectID, error)
}
```

`*hostsvc.Server` satisfies this interface. `buildCandidates` and `openWithPicker`
accept a `projectLister` instead of `codeRoot + resolver`.

**Rationale**: Per project conventions, interfaces are defined in the consumer package
to avoid the `ifaces/` anti-pattern. This also makes unit-testing `buildCandidates`
trivial (stub the interface).

## Risks / Trade-offs

- **Goroutine explosion on abnormally deep trees** → In practice `repositories/` is
  2–4 levels deep. If a user has a pathological layout, goroutine count could spike.
  Acceptable for now; a semaphore can be added later if needed.

- **`sync.Once` caches a scan error** → If the first scan fails (e.g. permissions),
  every subsequent `Projects()` call returns the same error for the process lifetime.
  Acceptable: the error is surfaced immediately; the user retries the whole command.

- **Result ordering is non-deterministic** → Goroutine fan-out does not preserve
  directory order. `buildCandidates` already de-duplicates with a `seen` map and
  attached projects are prepended separately, so ordering of the filesystem portion
  was never guaranteed.

## Migration Plan

Pure refactor — no config changes, no proto changes, no data migration. Rolling out
is a straight binary replacement.

## Open Questions

_(none)_
