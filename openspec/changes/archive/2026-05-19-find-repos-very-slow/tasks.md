# Tasks: find-repos-very-slow

## 1. layout.ScanRepos — parallel fan-out scan (cmd/swm)

- [x] 1.1 Write failing test for `(*Resolver).ScanRepos`: correct repo set returned for a
  temp code root with multiple hosts, orgs, and repos (including sibling repos and repos
  nested under deeper paths)
- [x] 1.2 Write failing test: sub-repositories (nested `.git`) are excluded
- [x] 1.3 Write failing test: host directories are not stat'd for `.git` (verify only
  children of host dirs get the `.git` stat, not the host dir itself)
- [x] 1.4 Implement `ScanRepos(ctx context.Context) ([]*pluginv1.ProjectID, error)` on
  `*Resolver` in `cmd/swm/internal/core/layout/scan.go` — `os.ReadDir` on hosts, then
  goroutine-per-child fan-out with `wg.Go`; each worker stats for `.git` first
- [x] 1.5 Confirm all 1.x tests pass

## 2. hostsvc.Server scan cache (cmd/swm)

- [x] 2.1 Write failing test: `Server.Projects()` called twice returns identical results
  and the underlying scan runs exactly once (instrument with a counter or use a fake
  `ScanRepos` that increments a call counter)
- [x] 2.2 Add `Projects(ctx context.Context) ([]*pluginv1.ProjectID, error)` to
  `hostsvc.Server` backed by `sync.Once` calling `resolver.ScanRepos`
- [x] 2.3 Refactor `hostsvc.Server.ListProjects` to stream from `s.Projects()` instead
  of its own `filepath.WalkDir`
- [x] 2.4 Confirm all 2.x tests pass and existing `TestListProjects_*` tests still pass

## 3. workspace.buildCandidates consumes projectLister (cmd/swm)

- [x] 3.1 Write failing test: `buildCandidates` with a stub `projectLister` returns
  attached projects first, then all repos from the lister, de-duplicated
- [x] 3.2 Define exported `ProjectLister` interface in
  `cmd/swm/internal/cli/workspace/open.go`:
  `Projects(ctx context.Context) ([]*pluginv1.ProjectID, error)`
- [x] 3.3 Replace the `filepath.WalkDir` body of `buildCandidates` with a call to
  `lister.Projects(ctx)` — update its signature to accept `(ctx, lister, st)`
- [x] 3.4 Update `openWithPicker` to accept and thread a `projectLister`
- [x] 3.5 Wire `*hostsvc.Server` as the `projectLister` at the call-site in the `open`
  cobra command handler (via `WithProjectLister` option threaded through `NewRootCmd`)
- [x] 3.6 Update `export_test.go` (`BuildCandidates`) to match the new signature
- [x] 3.7 Confirm all 3.x tests pass and existing `TestBuildCandidates_*` tests still pass

## 4. Verification (cmd/swm)

- [x] 4.1 Run `task fmt && task lint && task test` — all must exit 0
- [ ] 4.2 Manually time `swm workspace open _default` before and after on a real code
  root to confirm improvement (document delta in commit message)
