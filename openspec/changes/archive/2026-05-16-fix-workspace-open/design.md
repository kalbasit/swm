## Context

`cmd/swm/main.go` loads the config by calling `config.Load(os.Getenv("SWM_CONFIG"))`. When `$SWM_CONFIG` is unset the empty string is passed to `os.ReadFile("")`, which fails immediately; the error satisfies `os.IsNotExist`, so `ErrConfigNotFound` is returned and `main` falls back to `config.Defaults()` — a bare struct with no plugin names set. The user's `$XDG_CONFIG_HOME/swm/config.toml` is never consulted.

`config.Load` itself is correct; the gap is that `main` doesn't supply the XDG default path when `$SWM_CONFIG` is empty.

## Goals / Non-Goals

**Goals:**
- Make `swm` read `$XDG_CONFIG_HOME/swm/config.toml` automatically when `$SWM_CONFIG` is unset.
- Preserve the existing override behaviour: `$SWM_CONFIG` takes full precedence when set.
- Keep "file not found" a non-fatal condition (fall back to defaults), consistent with today.

**Non-Goals:**
- Config file merging or layered config (global + user + project).
- Detecting or auto-configuring any plugin (separate concern).
- Changes to `config.Load`, `config.Config`, or any plugin code.

## Decisions

### D1 — Resolve path in `main`, not in `config.Load`

`config.Load` takes an explicit path and is already unit-tested that way. Changing its signature to "use XDG when path is empty" would push environment-awareness into a package that is intentionally pure. Instead, `main` computes the effective path before calling `Load`:

```go
configPath := os.Getenv("SWM_CONFIG")
if configPath == "" {
    configPath = filepath.Join(xdg.ConfigHome, "swm", "config.toml")
}
cfg, err := config.Load(configPath)
```

Alternative considered: add an `AutoLoad()` function in the `config` package that does the XDG lookup. Rejected — it would add surface area and make `config` depend on XDG for a concern that belongs in the entry point.

### D2 — Preserve `ErrConfigNotFound` semantics

If the XDG default file doesn't exist, `config.Load` returns `ErrConfigNotFound` and `main` falls back to `config.Defaults()`. This is unchanged behaviour: users without any config file continue to get sensible defaults.

## Risks / Trade-offs

- **Behaviour change for users who rely on no-config defaults**: Any user who intentionally runs without a config file but happens to have an unrelated `~/.config/swm/config.toml` will now have it loaded. Low risk in practice (the directory name is swm-specific).
- **`xdg.ConfigHome` respects `$XDG_CONFIG_HOME`**: If the env var is set to a non-standard path, `swm` will look there. This is correct XDG behaviour.

## Migration Plan

No migration needed. The change is backward-compatible:
- Users without a config file: no change (still uses defaults).
- Users with `$SWM_CONFIG` set: no change (env var still wins).
- Users with `~/.config/swm/config.toml` and no env var: now works correctly.
