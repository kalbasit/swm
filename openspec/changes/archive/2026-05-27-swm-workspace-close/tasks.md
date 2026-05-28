## 1. Unit Tests (Red)

- [x] 1.1 `cmd/swm`: Write failing unit tests in `cmd/swm/internal/cli/workspace/close_test.go` covering all spec scenarios: close running workspace by name, no active workspace (idempotent success), no arg + `$SWM_STORY` set, no arg + `$SWM_STORY` unset (error), explicit arg overrides env, session plugin absent (error), `ListWorkspaces` error

## 2. Core Implementation (Green)

- [x] 2.1 `cmd/swm`: Create `cmd/swm/internal/cli/workspace/close.go` with `NewCloseCmd` — define the `pluginManager` interface (same shape as in `open.go`), implement the `ListWorkspaces`→filter-by-story-name→`CloseWorkspace` sequence with fatal error semantics and idempotent "no workspace found" success path
- [x] 2.2 `cmd/swm`: Add shell completion for `<name>` argument (list story names from store, matching pattern in `open.go` and `list.go`)
- [x] 2.3 `cmd/swm`: Update `cmd/swm/internal/cli/workspace/export_test.go` (if it exists) to export any new unexported helpers needed by tests

## 3. Wiring

- [x] 3.1 `cmd/swm`: Register `NewCloseCmd` on the workspace cobra sub-command group (locate where `NewOpenCmd` and `NewListCmd` are added and add `close` alongside them)

## 4. Verification

- [x] 4.1 Run `task fmt` and fix any formatting issues
- [x] 4.2 Run `task lint` and fix any lint issues
- [x] 4.3 Run `task test` and confirm all tests pass
