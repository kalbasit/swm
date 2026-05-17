## ADDED Requirements

### Requirement: Config file resolution order
When `swm` starts it SHALL resolve the configuration file path using the following precedence (first match wins):
1. `$SWM_CONFIG` environment variable, if set and non-empty.
2. `$XDG_CONFIG_HOME/swm/config.toml` (where `$XDG_CONFIG_HOME` defaults to `~/.config` per the XDG Base Directory Specification).

If the resolved file does not exist, `swm` SHALL start with built-in defaults and SHALL NOT treat a missing file as an error.

#### Scenario: SWM_CONFIG env var overrides XDG default
- **WHEN** `$SWM_CONFIG` is set to `/custom/path/config.toml` and that file exists
- **THEN** `swm` loads config from `/custom/path/config.toml`, ignoring `$XDG_CONFIG_HOME/swm/config.toml`

#### Scenario: XDG default used when SWM_CONFIG is unset
- **WHEN** `$SWM_CONFIG` is unset and `$XDG_CONFIG_HOME/swm/config.toml` exists with `[plugins] session = "tmux"`
- **THEN** `swm` loads config from the XDG path and plugin commands succeed

#### Scenario: Missing config file falls back to defaults
- **WHEN** `$SWM_CONFIG` is unset and `$XDG_CONFIG_HOME/swm/config.toml` does not exist
- **THEN** `swm` starts with built-in defaults (code_root=~/code, default_story=_default, no plugins) and exits zero
