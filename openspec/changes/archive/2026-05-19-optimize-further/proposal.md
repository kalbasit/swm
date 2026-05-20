## Why

Plugin subprocess startup accounts for ~0.7s of `swm workspace open`'s 1.5s
runtime: each capability (picker, session) is launched lazily and serially on
first `Manager.Get()` call. Commands know statically which capabilities they
need — pre-starting those in parallel at command entry eliminates the serial
startup chain without changing the lazy-by-default behavior for capabilities
that turn out not to be needed.

## What Changes

- Each cobra command that needs specific capabilities declares them via a new
  `WithEagerPlugins(capabilities ...string)` `PersistentPreRunE` hook wired
  through `NewRootCmd`.
- `Manager` gains a `Warm(ctx context.Context, capabilities ...string) error`
  method that starts all listed plugins concurrently (one goroutine per
  capability) and returns the first error, if any.
- `workspace open` declares `["picker", "session"]`; `workspace list` declares
  none.
- The existing lazy `Get()` path is unchanged — commands that don't call `Warm`
  start plugins on first use exactly as today.
- `Manager.Close()` is unchanged; it terminates all started processes regardless
  of how they were started.

## Non-goals

- Changing plugin discovery or binary resolution order.
- Warming plugins that are not statically known at compile time.
- Starting plugins that the running command does not actually use.
- Any changes to the plugin gRPC protocol or proto definitions.

## Capabilities

### New Capabilities

_(none)_

### Modified Capabilities

- `plugin-lifecycle`: the "Lazy launch" requirement gains a parallel eager
  startup path — `Warm()` pre-starts a declared set of capabilities
  concurrently before the command body runs, while preserving lazy behavior for
  everything else.

## Impact

- `cmd/swm/internal/pluginmgr/manager.go`: add `Warm` method.
- `cmd/swm/internal/cli/root.go`: wire a `PersistentPreRunE` that calls
  `mgr.Warm` with the capability set each sub-command declares.
- `cmd/swm/internal/cli/workspace/open.go`: declare `["picker", "session"]`.
- Potentially other command files that have known capability requirements.
- No proto changes.
