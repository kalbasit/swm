## 1. Tests (Red)

- [x] 1.1 `cmd/swm`: Add failing test cases for `ListProjects` in `hostsvc/server_test.go` and for `buildCandidates` in `cli/workspace/open_candidates_test.go` — fixtures with a top-level repo containing nested `.git` dirs assert only the top-level project is returned

## 2. Implementation (Green)

- [x] 2.1 `cmd/swm`: Fix both walk closures (`hostsvc/server.go` `ListProjects` and `cli/workspace/open.go` `buildCandidates`) to maintain a `projectRoots []string` slice; append the parent on `.git` discovery and return `filepath.SkipDir` for any directory whose path has a known root as a path-separator-terminated prefix

## 3. Verification

- [x] 3.1 Run `task fmt`, `task lint`, and `task test`; confirm all three exit with status 0
