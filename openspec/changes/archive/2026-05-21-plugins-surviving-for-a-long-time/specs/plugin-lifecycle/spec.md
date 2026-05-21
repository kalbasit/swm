## ADDED Requirements

### Requirement: Plugin manager interface exposes Close
The `pluginManager` interface consumed by any command that may call `exec`
SHALL include a `Close() error` method. This ensures the command can
explicitly terminate plugin subprocesses before replacing the process image.

#### Scenario: Close method present on pluginManager interface
- **WHEN** `workspace open` is compiled
- **THEN** the `pluginManager` interface in the `workspace` package includes `Close() error`

#### Scenario: Concrete manager satisfies extended interface
- **WHEN** `*pluginmgr.Manager` is assigned to a `pluginManager` variable
- **THEN** the assignment compiles without error (i.e. `Manager` already implements `Close()`)

### Requirement: Plugins are terminated before exec replaces the process
When `workspace open` is about to call `exec` to replace the swm process
with a terminal multiplexer client, it SHALL call `mgr.Close()` first.
No plugin subprocess SHALL outlive the `swm workspace open` invocation.

#### Scenario: No plugin processes remain after successful workspace open
- **WHEN** `swm workspace open` completes successfully and exec replaces the process
- **THEN** all plugin subprocesses spawned during that invocation have been terminated

#### Scenario: Close is called before exec
- **WHEN** `SwitchTo` returns an exec argv and `execFn` is about to be called
- **THEN** `mgr.Close()` is called before `execFn` is invoked

#### Scenario: Close error does not prevent exec
- **WHEN** `mgr.Close()` returns a non-nil error
- **THEN** the error is logged but does not prevent `execFn` from being called
- **THEN** the workspace open flow continues to exec

#### Scenario: Defer still covers non-exec error paths
- **WHEN** `workspace open` returns an error before reaching the exec path
- **THEN** the deferred `mgr.Close()` in `main.go` still terminates plugin subprocesses
