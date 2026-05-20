## 1. Refactor Manager.Warm (cmd/swm)

- [x] 1.1 Write failing tests in `cmd/swm/internal/pluginmgr/manager_test.go`: replace `TestWarm_ReturnsFirstError` with `TestWarm_ReturnsNilImmediately` asserting Warm always returns nil; update `TestWarm_StartsBothConcurrently` to assert Warm returns before plugins finish
- [x] 1.2 Write failing test `TestWarm_ErrorSurfacedByGet`: Warm a missing-binary capability, assert Warm returns nil, then assert Get returns non-nil error
- [x] 1.3 Rewrite `Manager.Warm` in `cmd/swm/internal/pluginmgr/manager.go`: remove WaitGroup/firstErr/setFirst/cancel; use `context.WithoutCancel(ctx)` for background goroutines; return nil immediately
- [x] 1.4 Confirm all manager tests pass (`task test`)

## 2. Simplify workspace open PreRunE (cmd/swm)

- [x] 2.1 Update `TestOpenCmd_PreRunE_WarmsPlugins` in `cmd/swm/internal/cli/workspace/open_test.go` to assert single Warm call with all three capabilities and nil error return from PreRunE
- [x] 2.2 Rewrite `workspace open PreRunE` in `cmd/swm/internal/cli/workspace/open.go`: remove WaitGroup, replace two Warm calls with one `mgr.Warm(ctx, "picker", "session", "vcs")`, return nil
- [x] 2.3 Confirm workspace open tests pass

## 3. Simplify clone and story remove PreRunE (cmd/swm)

- [x] 3.1 Update `TestCloneCmd_PreRunE_WarmsVCS` in `cmd/swm/internal/cli/clone_test.go` to expect PreRunE returns nil regardless of Warm outcome
- [x] 3.2 Update `cmd/swm/internal/cli/clone.go` PreRunE: call `mgr.Warm(ctx, "vcs")` and return nil
- [x] 3.3 Update `TestRemoveCmd_PreRunE_WarmsPlugins` in `cmd/swm/internal/cli/story/remove_test.go` to expect PreRunE returns nil
- [x] 3.4 Update `cmd/swm/internal/cli/story/remove.go` PreRunE: call `mgr.Warm(ctx, "vcs", "session")` and return nil
- [x] 3.5 Confirm clone and story remove tests pass

## 4. Final verification

- [x] 4.1 Run `task fmt` and apply any formatting changes
- [x] 4.2 Run `task lint` and fix any lint errors
- [x] 4.3 Run `task test` and confirm all tests pass
