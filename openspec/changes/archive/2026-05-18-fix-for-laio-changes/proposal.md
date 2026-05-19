## Why

laio-cli v0.17+ gates in-place session reconfiguration behind the explicit `--replace-current-session` flag. Before this change, passing `--tmux-socket` from inside tmux automatically reconfigured the current session; now it only targets the socket, and without `--replace-current-session` laio creates or switches to a named session instead — breaking every swm setup that relies on laio patching the active pane group session.

## What Changes

- Add `--replace-current-session` to all `pane_group_command` laio examples in `plugins/session-tmux/README.md`
- Update `testLaioPaneGroupCommandTOML` constant and `wantCmd` assertion in `plugins/session-tmux/internal/session/tmux_test.go` to include the new flag
- Update the laio scenario examples in `openspec/specs/session-tmux/spec.md` to include `--replace-current-session`
- Add `pane_group_command` with the updated laio invocation to `~/.config/swm/config.toml`

## Capabilities

### New Capabilities

_(none)_

### Modified Capabilities

- **session-tmux** — the documented and tested `pane_group_command` laio invocation changes; the spec scenarios for `OpenPaneGroup` with a custom command must reflect the new required flag.

## Impact

- `plugins/session-tmux/README.md` — two example `pane_group_command` lines updated
- `plugins/session-tmux/internal/session/tmux_test.go` — `testLaioPaneGroupCommandTOML` and the `wantCmd` in `TestOpenPaneGroup_WithPaneGroupCommand` updated
- `openspec/specs/session-tmux/spec.md` — two laio scenario blocks updated
- `~/.config/swm/config.toml` — `pane_group_command` entry added under `[plugins.config.session-tmux]`
- No proto changes; no new RPC surface; no breaking changes to the plugin API.
