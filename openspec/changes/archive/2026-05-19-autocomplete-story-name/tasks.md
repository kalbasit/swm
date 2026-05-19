## 1. Story name completion for `workspace open` (cmd/swm)

- [x] 1.1 Write failing unit tests in `cmd/swm/internal/cli/workspace/open_test.go` covering: story names returned on Tab, no-file-comp directive, and graceful store error
- [x] 1.2 Add `ValidArgsFunction` to `NewOpenCmd` in `cmd/swm/internal/cli/workspace/open.go` that calls `store.List` and returns story names with `cobra.ShellCompDirectiveNoFileComp`
- [x] 1.3 Confirm all `workspace open` completion unit tests pass

## 2. Story name completion for `story remove` (cmd/swm)

- [x] 2.1 Write failing unit tests in `cmd/swm/internal/cli/story/remove_test.go` covering: story names returned on Tab, no-file-comp directive, and graceful store error
- [x] 2.2 Add `ValidArgsFunction` to `NewRemoveCmd` in `cmd/swm/internal/cli/story/remove.go` that calls `store.List` and returns story names with `cobra.ShellCompDirectiveNoFileComp`
- [x] 2.3 Confirm all `story remove` completion unit tests pass

## 3. Verification

- [x] 3.1 Run `task fmt && task lint && task test` in `cmd/swm` — confirm all pass with zero errors
