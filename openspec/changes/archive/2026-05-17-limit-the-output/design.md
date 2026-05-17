## Context

`cmd/swm/internal/pluginmgr/manager.go` calls `goplugin.NewClient` in two places (capability plugins and forge plugins) without setting `ClientConfig.Logger`. When `Logger` is nil, go-plugin constructs its own `hclog` logger via `hclog.New(&hclog.LoggerOptions{Level: hclog.Trace})` (see go-plugin `client.go:418`), which defaults to TRACE level regardless of swm's `--log-level` flag. This leaks DEBUG/TRACE lines to stderr during every command invocation.

swm's own structured logging uses `log/slog` with a configurable level (default `warn`) set in `root.go`. The `Manager` currently has no knowledge of the active log level.

## Goals / Non-Goals

**Goals:**
- go-plugin's internal hclog logger respects the same level as swm's `--log-level` flag.
- Both `goplugin.NewClient` call sites in `manager.go` are fixed consistently.
- The fix is contained to `pluginmgr` and `cli/root.go`; no other packages change.

**Non-Goals:**
- Routing go-plugin's hclog lines through swm's slog pipeline (structural bridging).
- Exposing a separate flag to control go-plugin's internal logger independently.
- Silencing go-plugin WARN/ERROR messages at the default `warn` level (those should remain visible).

## Decisions

### D1 — Use `hclog.New` with a derived level rather than `hclog.NewNullLogger()`

**Decision**: Configure an `hclog.Logger` with a level translated from the slog level, not a null logger.

**Rationale**: A null logger would suppress legitimate WARN/ERROR go-plugin messages (e.g., plugin process crash, handshake failure) that are useful even at the default `warn` level. Translating the level preserves this signal.

**Alternatives considered**:
- `hclog.NewNullLogger()`: simplest but silences all go-plugin internal messages, hiding failures.
- Bridge hclog → slog: correct long-term but adds complexity and an adapter type; not needed now.

**Level mapping** (`slog` → `hclog`):
| slog         | hclog         |
|--------------|---------------|
| LevelDebug   | hclog.Debug   |
| LevelInfo    | hclog.Info    |
| LevelWarn    | hclog.Warn    |
| LevelError   | hclog.Error   |
| (anything else / unset) | hclog.Warn (safe default) |

---

### D2 — Add `WithLogLevel(slog.Level)` option to `Manager`

**Decision**: Store the slog level on `Manager` and derive the hclog logger inside a shared `newClientConfig` helper that builds `goplugin.ClientConfig`.

**Rationale**: This follows the existing `WithStderr` option pattern, keeps construction logic in one place, and avoids threading the level through every call site individually. A shared helper removes the duplication between the capability and forge `NewClient` call sites.

**Alternative considered**: Pass `hclog.Logger` directly via `WithHCLogger`. Rejected because it couples the caller to the `hclog` package, which is an implementation detail of `pluginmgr`.

---

### D3 — Wire `--log-level` through to `Manager` in `root.go`

**Decision**: After resolving the slog level from the flag string, call `pluginmgr.WithLogLevel(level)` when constructing the manager in the root command's `PersistentPreRunE`.

**Rationale**: The parsed level is already available at that point; no new plumbing is required.

## Risks / Trade-offs

- **hclog import in `pluginmgr`**: `manager.go` already imports `goplugin`, which transitively provides `hclog`. Using `hclog.New` adds a direct import — acceptable since `pluginmgr` is already go-plugin-specific.
- **Level mismatch at default**: At the default `warn` level, go-plugin WARN messages will appear (e.g., plugin restart warnings). This is intentional and correct.

## Migration Plan

No data migration. The change is purely additive to `Manager`'s API (`WithLogLevel` is optional; if omitted, hclog defaults to `hclog.Warn` — better than the current TRACE default). No rollback needed.

## Open Questions

_None._
