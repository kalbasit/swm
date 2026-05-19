## Context

`cmd/swm/internal/config` currently provides read-only access: `Load` reads and unmarshals `config.toml`; `Defaults` returns zero-file defaults. There is no write path and no key registry. Users must hand-edit the TOML file to change any setting.

The `swm config` command adds `get`, `set`, and `list` subcommands to `cmd/swm`. It is host-only: no plugin protocol changes, no proto bump.

## Goals / Non-Goals

**Goals:**
- `swm config get <key>` — print the effective value of any registered scalar key (configured or default).
- `swm config set <key> <value>` — write a new scalar value to the config file, creating it if absent.
- `swm config list` — print only file-configured key/value pairs.
- `swm config list --all` — print all registered keys with their effective values.
- A static key registry that is the single source of truth for key names, types, defaults, and descriptions.

**Non-Goals:**
- Setting array fields (`plugins.forges`) or nested plugin config (`plugins.config.*`) — get/list display them, set is scalar-only in v1.
- Preserving TOML comments on write — acceptable loss.
- Interactive TUI editor or `swm config edit`.
- Per-plugin config namespacing under `plugins.config.*` as settable keys.

## Decisions

### 1. Key notation: dot-separated TOML path

Keys use dot-separated segments mirroring the TOML hierarchy:

| Key | TOML field |
|---|---|
| `code_root` | `code_root` |
| `default_story` | `default_story` |
| `plugins.session` | `[plugins] session` |
| `plugins.vcs` | `[plugins] vcs` |
| `plugins.picker` | `[plugins] picker` |
| `plugins.forges` | `[plugins] forges` (read-only in v1) |
| `plugins.paths.<name>` | `[plugins.paths] <name>` |
| `story.branch_name_template` | `[story] branch_name_template` |

**Alternatives considered:** `swm config set plugins session tmux` (positional segments) — rejected because it makes scripting harder and doesn't compose with shell quoting conventions.

### 2. Key registry in `config/keys.go`

A static `[]KeyDef` slice lives in `cmd/swm/internal/config/keys.go`. Each entry carries: dot-path, human description, whether it is writable via `set`, and a getter/setter func pair that operates on `*Config`. This is the single source of truth for `list --all` ordering, `get` validation, and `set` routing.

**Alternatives considered:** Reflection over struct tags — rejected because it cannot express write-only vs read-only, cannot handle map sub-keys (`plugins.paths.<name>`), and produces unpredictable key ordering.

### 3. Write strategy: unmarshal → mutate → marshal → write

`set` flow:
1. If config file does not exist, create parent dirs and write an empty file first.
2. Read raw bytes.
3. Unmarshal into `Config` via `toml.Unmarshal`.
4. Apply the mutation through the key's setter.
5. Marshal with `toml.Marshal`.
6. Write back atomically (write to `<path>.tmp`, then `os.Rename`).

TOML comments are lost on round-trip — this is documented and accepted. `go-toml/v2` has no comment-preserving round-trip API.

**Alternatives considered:** Regex/sed-style in-place line replacement — rejected because it is fragile against multi-line values, array syntax, and inline tables.

### 4. `get` returns effective value (default or configured)

`get <key>` always returns a value (never empty) because it reads the effective config (defaults merged with file). This is consistent with how every other `swm` command uses config.

### 5. `list` vs `list --all`

`list` (no flag): reads raw TOML file, emits only keys present in the file. If no file exists, prints nothing and exits 0.
`list --all`: iterates the key registry in definition order, emitting effective values for all registered keys.

Both output `key = value` lines (TOML scalar syntax), one per line.

## Risks / Trade-offs

- **Comment loss on write** → Documented in command help text. Users who care about comments should use `$EDITOR`.
- **Partial key coverage** — `plugins.forges` and `plugins.config.*` are display-only in v1 → Mitigation: `set` returns a clear error "key is not writable in this version" rather than silently doing nothing.
- **Concurrent writes** — no file locking on config writes → Mitigation: atomic rename reduces the window; config writes are rare interactive actions, not high-frequency operations. File locking can be added later if needed.
- **`plugins.paths.<name>` dynamic keys** — the key registry must handle open-ended map sub-keys → Mitigation: registry uses a wildcard entry `plugins.paths.*` with a dynamic getter/setter.

## Open Questions

- Should `swm config set` create the config file if it does not exist, or require the user to create it first? **Decision: create it** (mkdir -p + empty file, then write).
- Output format for `list --all` on array values (`plugins.forges`): TOML inline array (`["tmux"]`) or one-per-line? **Decision: TOML inline array** for consistency with the file format.
