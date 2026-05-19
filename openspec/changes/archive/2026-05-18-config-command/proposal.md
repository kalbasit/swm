## Why

Users must manually edit `~/.config/swm/config.toml` to change any swm setting, with no way to read or write values from the command line. A `swm config` command makes configuration scriptable and removes the need to know the TOML file path or format.

## What Changes

- Add `swm config get <key>` — print the current value of a single config key to stdout.
- Add `swm config set <key> <value>` — write a new value for a config key to the config file.
- Add `swm config list` — print only explicitly configured key-value pairs in dot-notation (e.g. `plugins.session = tmux`).
- Add `swm config list --all` — print all available keys with their effective values (configured or default).
- Add config write support to `cmd/swm/internal/config` (currently read-only).

## Capabilities

### New Capabilities

- `config-command` — CLI subcommands for reading and writing swm configuration without touching the TOML file directly.

### Modified Capabilities

<!-- No existing spec-level requirements change. -->

## Non-goals

- No interactive TUI editor.
- No validation of plugin-specific config values under `plugins.config.*` — those are opaque to the host.
- No merging or diffing of config files.
- No `swm config edit` (opening `$EDITOR`) — out of scope for now.
- No support for writing array fields (e.g. `plugins.forges`) in v1 of this command; get and list will display them, set will be limited to scalar and map-string values.

## Impact

- `cmd/swm` — new `config` cobra command with `get`, `set`, and `list` subcommands.
- `cmd/swm/internal/config` — new write path: load → mutate → marshal TOML → write back to the resolved config file path.
- Capability surfaces affected: **none** (host-only CLI feature, no plugin protocol changes).
- No proto changes required.
