# Design: swm-with-plugins-development

## Context

Plugin discovery today follows a fixed three-tier order: explicit `config.toml`
paths → XDG data dir → PATH.  There is no way to inject dev-built binaries ahead
of system-installed ones without either modifying config or manipulating PATH.

The current dev workflow requires `nix profile remove swm-full && nix profile
install .#swm-full` on every change — several minutes per iteration.  Development
happens inside per-story git worktrees whose absolute paths are not stable, so
any solution relying on a fixed installed path is ruled out.

## Goals / Non-Goals

**Goals**
- Let dev-built plugin binaries win over system-installed ones without config changes.
- Provide a single script (`scripts/swm-dev`) that builds and runs swm from source
  in one command, usable both inside and outside direnv.
- Keep the mechanism general enough to be useful in CI or integration tests.

**Non-Goals**
- Plugin sandboxing or version pinning.
- Any changes to proto, SDK, or plugin RPC protocol.
- A `swm dev` subcommand inside the swm binary.

## Decisions

### 1. Env var (`SWM_PLUGIN_PATH`) over flag or config key

A flag would require the caller to pass it on every invocation; the `swm-dev`
script would need to forward it through `exec`, and any tool that shells out to
`swm` would break.  An env var propagates automatically across `exec` chains.  A
config key would persist on disk and risk leaking into non-dev sessions.

### 2. `SWM_PLUGIN_PATH` is tier 0 — before explicit config paths

The purpose is to guarantee dev binaries win unconditionally.  Placing the env
var after explicit config paths would mean a stale config entry could silently
shadow the dev binary, creating hard-to-debug situations.

### 3. Colon-separated list, left-to-right, non-existent entries silently skipped

Consistent with `PATH` convention.  Skipping missing dirs allows partial
`SWM_PLUGIN_PATH` entries without erroring (e.g. when only some plugins are
built locally).

### 4. `scripts/swm-dev` always rebuilds — no freshness check

A `make`-style mtime comparison adds code complexity for modest gain.  Go's
incremental builds already skip unchanged packages; a clean rebuild of an
unchanged tree completes in ~100 ms.  Always-rebuild eliminates stale-binary
bugs and keeps the script trivially auditable.

### 5. Script discovers repo root via `git rev-parse --show-toplevel`

Works correctly from any directory inside any worktree regardless of the
absolute path.  No hardcoded paths, no environment variables required.

### 6. Devshell adds `$PWD/scripts` to PATH via `shellHook`

`$PWD` in a Nix `mkShell` `shellHook` is the project root at the moment direnv
activates.  This makes `swm-dev` available without a path prefix inside the
devshell while keeping the approach simple (no derivation, no wrapper package).
Outside direnv, `./scripts/swm-dev` is used directly.

## Risks / Trade-offs

- **`SWM_PLUGIN_PATH` set accidentally** → silently no-ops on non-existent dirs;
  worst case is a missing plugin error, same as today.  Mitigation: document
  clearly that this var is for development only.
- **Build failure blocks invocation** → `set -euo pipefail` in the script means a
  compile error surfaces immediately with a clear Go error message.  This is
  desirable, not a risk.
- **`go work` not initialised** → `go build` may fail if `go.work` is absent.
  The devshell `shellHook` already ensures `go work init` and `go work use` run,
  so this is only a concern when outside direnv.  The script should check for
  `go.work` and call `go work use` if needed, or document the prerequisite.

## Migration Plan

Fully additive — no existing behaviour changes.  `SWM_PLUGIN_PATH` unset (the
common case) leaves discovery identical to today.  No rollback needed.

## Open Questions

None.
