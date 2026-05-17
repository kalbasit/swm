# Proposal: fix-workspace-open

## Why

`swm workspace open` (and all plugin-dependent commands) fail with "no session plugin configured" even when `~/.config/swm/config.toml` is properly populated. The root cause is that `main.go` loads the config exclusively from the `$SWM_CONFIG` environment variable and silently falls back to bare defaults when the variable is unset — it never tries the XDG standard path (`$XDG_CONFIG_HOME/swm/config.toml`).

## What Changes

- `cmd/swm/main.go`: when `$SWM_CONFIG` is empty, default the config path to `filepath.Join(xdg.ConfigHome, "swm", "config.toml")` before calling `config.Load`. `ErrConfigNotFound` is still handled gracefully (defaults applied) so a missing config file is not an error.

**Non-goals**

- Changing the `config.Load` API.
- Supporting multiple config file locations or config merging.
- Any changes to plugin resolution logic.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `workflow-commands` — the implicit contract that commands respect `~/.config/swm/config.toml` was already assumed by users but not enforced by the code. A new scenario documents that `$SWM_CONFIG` overrides the XDG default and that the XDG default is used when the variable is unset.

## Impact

- `cmd/swm/main.go`: 2-line change — compute default config path from `xdg.ConfigHome` before calling `config.Load`.
- `openspec/specs/workflow-commands/spec.md`: add a requirement describing config file resolution order (`$SWM_CONFIG` → XDG default → built-in defaults).
- Capability surface affected: **none** (this is startup/bootstrapping, not a plugin capability).
- No proto changes required.
