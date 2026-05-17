## Why

When a user runs any `swm` command, the terminal is flooded with `[DEBUG]` and `[TRACE]` lines from go-plugin's internal logger even though swm's default `--log-level` is `warn`. The go-plugin library (`github.com/hashicorp/go-plugin`) uses its own `hclog` logger that defaults to DEBUG level; because `Manager` never passes a `Logger` to `goplugin.ClientConfig`, go-plugin ignores swm's slog level entirely.

## What Changes

- Add a `WithLogLevel` option to `pluginmgr.Manager` that accepts a `slog.Level` and stores it.
- When constructing each `goplugin.ClientConfig`, derive an `hclog.Level` from the stored slog level and pass an appropriately configured `hclog.Logger` as `ClientConfig.Logger`.
- This affects both `goplugin.NewClient` call sites in `manager.go` (lines 161 and 287).
- Wire the root command's `--log-level` flag value through to `Manager` via `WithLogLevel`.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

_None._ The plugin-lifecycle spec does not specify requirements for the internal go-plugin logger level; this is a host-side implementation bug fix with no behavioral contract changes.

## Non-goals

- Forwarding go-plugin internal log lines into swm's slog pipeline (bridging hclog → slog). The goal is suppression at the right level, not structured re-emission.
- Changing any plugin protocol or proto definitions.
- Exposing a separate flag to control go-plugin's logger independently of `--log-level`.

## Impact

- **Affected code**: `cmd/swm/internal/pluginmgr/manager.go` (both `goplugin.NewClient` call sites), `cmd/swm/internal/cli/root.go` (wiring `WithLogLevel`).
- **Capability surface**: `none` — internal to the host, no plugin protocol changes.
- **Proto changes**: none.
- **Dependencies**: `github.com/hashicorp/go-plugin` already provides `hclog`; no new dependencies.
