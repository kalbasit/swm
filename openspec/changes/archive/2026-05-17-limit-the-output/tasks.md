## 1. Manager — level field and option

- [x] 1.1 Add `logLevel hclog.Level` field to `Manager` struct and import `github.com/hashicorp/go-hclog` (`cmd/swm/internal/pluginmgr/manager.go`)
- [x] 1.2 Implement `WithLogLevel(l slog.Level) Option` that maps slog → hclog level and stores it on `Manager` (`cmd/swm/internal/pluginmgr/manager.go`)
- [x] 1.3 Set `Manager.logLevel` default to `hclog.Warn` in `New()` so the zero value is safe (`cmd/swm/internal/pluginmgr/manager.go`)

## 2. Manager — shared client config helper

- [x] 2.1 Extract a `buildClientConfig(pluginCmd *exec.Cmd, set goplugin.PluginSet) *goplugin.ClientConfig` method on `Manager` that builds the config with `Logger` set to an `hclog.New` at `m.logLevel` (`cmd/swm/internal/pluginmgr/manager.go`)
- [x] 2.2 Replace the inline `goplugin.ClientConfig` literal in `launchPlugin` (line ~161) with a call to `buildClientConfig` (`cmd/swm/internal/pluginmgr/manager.go`)
- [x] 2.3 Replace the inline `goplugin.ClientConfig` literal in `loadForges` (line ~287) with a call to `buildClientConfig` (`cmd/swm/internal/pluginmgr/manager.go`)

## 3. CLI wiring

- [x] 3.1 After resolving the slog level from `--log-level` in `PersistentPreRunE`, pass `pluginmgr.WithLogLevel(level)` when constructing the `Manager` (`cmd/swm/internal/cli/root.go`)

## 4. Tests

- [x] 4.1 Unit test: `WithLogLevel` maps slog.LevelDebug/Info/Warn/Error to hclog.Debug/Info/Warn/Error respectively; unmapped values default to hclog.Warn (`cmd/swm/internal/pluginmgr/manager_test.go`)
- [x] 4.2 Integration test: at default level (`warn`), launching a real plugin binary produces no `[DEBUG]` or `[TRACE]` lines on the captured stderr writer (`cmd/swm/internal/pluginmgr/manager_test.go`)
- [x] 4.3 Integration test: at `debug` level, launching a real plugin binary produces at least one `[DEBUG]` line on the captured stderr writer (`cmd/swm/internal/pluginmgr/manager_test.go`)
