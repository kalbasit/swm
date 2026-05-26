## Why

The `session-tmux` plugin currently creates one bare tmux session per project but does no window or pane layout — users rely on Laio (a separate Rust binary) to lay out windows and panes from a YAML config. Laio's author is unwilling to make the integration changes needed for SWM, creating an unresolvable external dependency on a tool we cannot influence.

## What Changes

- **NEW**: Two-tier layout config (global at `$XDG_CONFIG_HOME/swm/session-tmux.toml`, per-repo at `.swm/session-tmux.toml`), both TOML and using the same schema. Per-repo wins; global is the user-wide default. Same tier model as hooks.
- **NEW**: `plugins/session-tmux` reads that config file during `OpenPaneGroup` and applies the layout to the newly created tmux session.
- **NEW**: Layout supports: named windows, nested pane splits with `flex`/`flex_direction` (row/column), per-window/pane working directories, startup commands, environment variables, and focus/zoom flags.
- **NEW**: Go text/template variable substitution in the config (replaces Laio's Tera template support), with `worktree_path` and `story_name` auto-injected.
- **REMOVE** (optional/gradual): Laio as a hard runtime requirement for users who relied on it via external hooks.

No proto changes are required. The plugin resolves and applies the config internally without any new gRPC surface.

## Capabilities

### New Capabilities

- `session-tmux-layout` — window/pane layout management built into the `session-tmux` plugin. Covers: config parsing, flex-split calculation, tmux command sequencing (new-window, split-window, resize-pane, send-keys). Lives in a new sub-package `plugins/session-tmux/internal/layout/`.

### Modified Capabilities

- `session-tmux` (`openspec/specs/session-tmux/spec.md`) — `OpenPaneGroup` now applies the layout config after session creation if `.swm/session-tmux.toml` exists. Behavior is additive: absence of the config file leaves existing behavior unchanged.

## Impact

- **Code**: `plugins/session-tmux/internal/session/tmux.go` (`OpenPaneGroup` extended); new package `plugins/session-tmux/internal/layout/` (~400–600 lines).
- **Config**: New optional file `.swm/session-tmux.toml` per repository (not global config).
- **Dependencies**: No new Go modules; uses `pelletier/go-toml/v2` (already a workspace dep) and `text/template` (stdlib).
- **Nix hashes**: `vendorHash` in `nix/packages/swm-plugin-session-tmux/default.nix` will need updating only if a new dep is added (unlikely).
- **Laio**: No code change to Laio. Users who migrate get the same feature set from SWM directly.
- **Non-goals**: Zellij support, serializing an existing tmux session to TOML, template variable arrays (Laio supports `--var k=v` repeated; SWM's first pass will use a flat string map only), GUI pane preview.
