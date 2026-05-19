# Proposal: swm-with-plugins-development

## Why

During development of swm or any of its plugins, there is no way to direct swm
to load locally-built plugin binaries — the host always discovers plugins from
PATH and XDG dirs, so a system-installed swm wins over a dev build and vice
versa.  This blocks iterative plugin development without hacking PATH or
installing binaries system-wide.

## What Changes

- Introduce a `SWM_PLUGIN_PATH` environment variable: a colon-separated list of
  directories that is **prepended** to the normal plugin search order (XDG dirs,
  then PATH).  Plugins found in `SWM_PLUGIN_PATH` take precedence over all
  system-level discoveries.
- Document `SWM_PLUGIN_PATH` in shell-completion help text and the existing
  plugin-lifecycle spec.
- Add a `scripts/swm-dev` script committed to the repository.  On each invocation
  it:
  1. Builds `cmd/swm` and all plugins under `plugins/` with `go build` into
     `$REPO_ROOT/.dev-bin/`.
  2. Sets `SWM_PLUGIN_PATH` to that directory.
  3. `exec`s the freshly built `swm` binary with all original arguments.
  This replaces the current workflow of `nix profile remove swm-full; nix profile
  install .#swm-full` (several minutes) with a single `swm-dev` invocation backed
  by Go's incremental builds (~100 ms when nothing changed).
- The Nix devshell adds `$REPO_ROOT/scripts` to `PATH` so `swm-dev` is available
  without a path prefix when inside direnv.  Outside direnv (e.g. running tests
  in a separate shell while still inside the worktree), invoke it directly as
  `./scripts/swm-dev`.  Because development happens inside git worktrees whose
  paths change per story, no stable installed path is assumed.
- No new flags, no new config keys, no new subcommands in the swm binary itself.

## Non-goals

- Plugin sandboxing or version pinning.
- A `swm dev` subcommand baked into the swm binary.
- Changing how plugins are discovered outside of `SWM_PLUGIN_PATH`.

## Capabilities

**New Capabilities**: none

**Modified Capabilities**:
- `plugin-lifecycle` — the plugin discovery algorithm gains a new first-priority
  search tier driven by `SWM_PLUGIN_PATH`.  Requirements change: the host MUST
  iterate each directory in `SWM_PLUGIN_PATH` before consulting XDG dirs or
  PATH.  A delta spec is needed.

## Impact

- `cmd/swm`: plugin loader / discovery code reads `SWM_PLUGIN_PATH` at startup.
- `openspec/specs/plugin-lifecycle/spec.md`: discovery algorithm section updated
  via delta spec in this change.
- `scripts/swm-dev`: new committed script, works from any shell inside the worktree.
- Nix devshell configuration: adds `$REPO_ROOT/scripts` to `PATH`.
- No proto changes; no new capabilities; no SDK changes.

Capability surface affected: **none** (internal host behavior only).
