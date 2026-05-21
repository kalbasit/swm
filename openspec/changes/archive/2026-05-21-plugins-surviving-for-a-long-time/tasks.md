## 1. Spec — delta spec for plugin-lifecycle (cmd/swm)

- [x] 1.1 Write failing test in `open_test.go` asserting `pluginManager` interface includes `Close() error`
- [x] 1.2 Write failing test asserting `mgr.Close()` is called before `execFn` in the happy path

## 2. Interface (cmd/swm)

- [x] 2.1 Add `Close() error` to the `pluginManager` interface in `cmd/swm/internal/cli/workspace/open.go`
- [x] 2.2 Update the fake/stub `pluginManager` in `open_test.go` to implement `Close() error`

## 3. Exec-path cleanup (cmd/swm)

- [x] 3.1 In `open.go`, call `mgr.Close()` immediately before `execFn(argv[0], argv, os.Environ())` in the exec branch; log but do not return on error
- [x] 3.2 Confirm existing `defer mgr.Close()` in `main.go` remains untouched (safety net for error paths)

## 4. Verification

- [x] 4.1 Run `task fmt && task lint && task test` — all must pass with exit 0
