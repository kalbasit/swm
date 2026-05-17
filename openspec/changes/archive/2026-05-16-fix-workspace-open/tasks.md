## 1. Fix config path resolution (cmd/swm)

- [x] 1.1 In `cmd/swm/main.go`, compute the effective config path: use `$SWM_CONFIG` when set, otherwise `filepath.Join(xdg.ConfigHome, "swm", "config.toml")`
- [x] 1.2 Add a unit test in `cmd/swm/internal/config/` (or an integration test) covering the three scenarios: `$SWM_CONFIG` override, XDG default found, XDG default missing → falls back to defaults

## 2. Spec update (openspec)

- [x] 2.1 Merge the delta spec (`specs/workflow-commands/spec.md`) into `openspec/specs/workflow-commands/spec.md` — add the "Config file resolution order" requirement and its three scenarios
