## 1. Key Registry (cmd/swm)

- [x] 1.1 Create `cmd/swm/internal/config/keys.go` — define `KeyDef` struct (dot-path, description, writable bool, getter/setter funcs on `*Config`) and the static registry covering all scalar keys: `code_root`, `default_story`, `plugins.session`, `plugins.vcs`, `plugins.picker`, `plugins.forges` (read-only), `plugins.paths.*` (wildcard), `story.branch_name_template`
- [x] 1.2 Write table-driven unit tests in `cmd/swm/internal/config/keys_test.go` — verify each registered key round-trips through its getter/setter, and that unknown keys return an error from the lookup function

## 2. Config Write Path (cmd/swm)

- [x] 2.1 Create `cmd/swm/internal/config/write.go` — implement `Save(path string, cfg *Config) error`: marshal `cfg` with `toml.Marshal`, write to `<path>.tmp`, then `os.Rename` to `path`; create parent directories if absent
- [x] 2.2 Write unit tests in `cmd/swm/internal/config/write_test.go` — cover: save to new path (creates dirs + file), save over existing file, marshal+reload round-trip preserves values

## 3. `swm config get` (cmd/swm)

- [x] 3.1 Create `cmd/swm/internal/cli/config/get.go` — implement `NewGetCmd(cfg *config.Config)` cobra command: look up key in registry, print effective value to stdout; exit non-zero with error listing valid keys for unknown key; require exactly one argument
- [x] 3.2 Write unit tests in `cmd/swm/internal/cli/config/get_test.go` — cover: known key with default (no file), known key configured in file, map sub-key, unknown key exits non-zero, missing argument exits non-zero

## 4. `swm config set` (cmd/swm)

- [x] 4.1 Create `cmd/swm/internal/cli/config/set.go` — implement `NewSetCmd(cfgPath string)` cobra command: reject non-writable and unknown keys with non-zero exit; load config (or start from `Defaults()` if file absent); apply setter; call `Save`; require exactly two arguments
- [x] 4.2 Write unit tests in `cmd/swm/internal/cli/config/set_test.go` — cover: set creates new file, set updates existing file, set persists value readable by `Load`, non-writable key exits non-zero, unknown key exits non-zero, wrong argument count exits non-zero

## 5. `swm config list` (cmd/swm)

- [x] 5.1 Create `cmd/swm/internal/cli/config/list.go` — implement `NewListCmd(cfgPath string, cfg *config.Config)` cobra command with `--all` bool flag; without `--all`: read raw file and emit only present keys in `key = value` format; with `--all`: iterate registry in definition order, emit effective values for every key; array values use TOML inline array format
- [x] 5.2 Write unit tests in `cmd/swm/internal/cli/config/list_test.go` — cover: no file → empty output, some configured keys → only those shown, `--all` with no file → all defaults shown, `--all` with partial file → mix of configured and default values, output order is stable

## 6. Wire config command into root (cmd/swm)

- [x] 6.1 Create `cmd/swm/internal/cli/config/cmd.go` — implement `NewConfigCmd(cfgPath string, cfg *config.Config)` that builds the `config` cobra command and adds `get`, `set`, and `list` as subcommands
- [x] 6.2 Update `cmd/swm/internal/cli/root.go` — add `cfgPath string` parameter to `NewRootCmd` and register `NewConfigCmd` as a subcommand of root
- [x] 6.3 Update `cmd/swm/main.go` — compute `cfgPath := config.ResolveConfigPath(...)` before calling `config.Load`, and pass it to `cli.NewRootCmd`

## 7. Verification

- [x] 7.1 Run `task fmt && task lint && task test` in `cmd/swm` — confirm zero exit status and all tests green
