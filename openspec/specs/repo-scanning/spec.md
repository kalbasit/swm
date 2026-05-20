# Spec: repo-scanning

## ADDED Requirements

### Requirement: Host enumerates repos once per invocation
The host SHALL scan `$CODE_ROOT/repositories/` at most once per process invocation and
cache the result. Subsequent consumers (project picker, `ListProjects` gRPC callers)
SHALL receive the cached result without triggering additional filesystem traversal.

#### Scenario: Second caller gets cached result
- **WHEN** `Projects()` is called twice on the same `hostsvc.Server` instance
- **THEN** the filesystem is walked exactly once and both calls return identical results

---

### Requirement: Repo scan uses parallel fan-out
The scan SHALL fan out one goroutine per subdirectory rather than walking serially.
Each worker SHALL check for `.git` before descending, and SHALL NOT descend into a
directory once `.git` is found at its root.

#### Scenario: Repos discovered correctly
- **WHEN** `$CODE_ROOT/repositories/` contains N repos at arbitrary nesting depths
- **THEN** `ScanRepos` returns exactly those N `ProjectID` values and no others

#### Scenario: Sub-repositories are excluded
- **WHEN** a repo contains nested `.git` directories (e.g. vendored modules, temp clones)
- **THEN** only the outermost repo is returned; nested `.git` dirs are not descended into

---

### Requirement: Host directories are never treated as repos
The scan SHALL skip the `.git` check for first-level entries under `repositories/`
(host directories such as `github.com`). A valid `ProjectID` requires host plus at
least one path segment, so a host directory itself can never be a repo root.

#### Scenario: Host directory with no repos
- **WHEN** `repositories/github.com/` exists but contains no repos with `.git`
- **THEN** no `ProjectID` entries are returned for that host

#### Scenario: Host directory skips .git stat
- **WHEN** `ScanRepos` is invoked
- **THEN** no `.git` stat is performed directly inside `repositories/` (only inside
  host subdirectories and deeper)

---

### Requirement: buildCandidates consumes the shared scan result
The `buildCandidates` function in the workspace CLI SHALL obtain its repo list from the
`hostsvc.Server` cache via the `projectLister` interface rather than performing its own
filesystem walk.

#### Scenario: Candidates include all on-disk repos
- **WHEN** `buildCandidates` is called with a `projectLister` backed by a code root
  containing repos
- **THEN** all repos returned by `projectLister.Projects()` appear in the candidate list

#### Scenario: Attached projects appear before filesystem repos
- **WHEN** the current story has attached projects
- **THEN** those projects appear first in the candidate list, followed by on-disk repos
