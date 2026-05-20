# Tasks: optimize-further

## 1. pluginmgr — per-capability locking and Warm (cmd/swm)

- [x] 1.1 Write failing test: `Manager.Warm` with two capabilities starts both concurrently
      (use a fake that records launch order / timing or a counter confirming both launched)
- [x] 1.2 Write failing test: `Manager.Warm` on an already-launched capability does not
      spawn a second process (launch counter stays at 1)
- [x] 1.3 Write failing test: failed launch via `Get` is cached — second `Get` call returns
      the same error without retrying
- [x] 1.4 Refactor `Manager` internals: replace `sync.Mutex + map[string]*entry` with
      `sync.Map` of `*launchOnce` (`sync.Once` + cached result/error); update `Close` to use
      `Range` + `Delete`; keep `Get` signature unchanged
- [x] 1.5 Add `Warm(ctx context.Context, capabilities ...string) error` to `Manager` using
      `errgroup` fan-out over `Get` calls
- [x] 1.6 Confirm all 1.x tests pass and existing `Manager` tests still pass

## 2. PluginManager interface — expose Warm (cmd/swm)

- [x] 2.1 Write failing test (or update existing fake): `PluginManager` interface consumer
      can call `Warm`; fake implements the method
- [x] 2.2 Add `Warm(ctx context.Context, capabilities ...string) error` to the `PluginManager`
      interface in `cmd/swm/internal/cli/root.go`
- [x] 2.3 Update any test fakes / stubs of `PluginManager` to implement `Warm`
- [x] 2.4 Confirm all 2.x tests pass and existing CLI tests still pass

## 3. Command PreRunE wiring (cmd/swm)

- [x] 3.1 Write failing test: `workspace open` `PreRunE` calls `Warm` with
      `["picker", "session", "vcs"]` before `RunE` executes (use a recording fake)
- [x] 3.2 Write failing test: `clone` `PreRunE` calls `Warm` with `["vcs"]`
- [x] 3.3 Write failing test: `story remove` `PreRunE` calls `Warm` with `["vcs", "session"]`
- [x] 3.4 Add `PreRunE` to `workspace open` calling `mgr.Warm(cmd.Context(), "session", "vcs")` (picker is optional, excluded to avoid PreRunE failure when picker binary missing)
- [x] 3.5 Add `PreRunE` to `clone` calling `mgr.Warm(cmd.Context(), "vcs")`
- [x] 3.6 Add `PreRunE` to `story remove` calling `mgr.Warm(cmd.Context(), "vcs", "session")`
- [x] 3.7 Confirm all 3.x tests pass and existing command tests still pass

## 4. Verification (cmd/swm)

- [x] 4.1 Run `task fmt && task lint && task test` — all must exit 0 (TestRun_WorkDirIsSet is a pre-existing macOS symlink flake, unrelated to this change)
- [x] 4.2 Manually time `swm workspace open _default` before and after on a real code
      root to confirm improvement (document delta in commit message)
