## Why

The laio integration was implemented (see archived change `2026-05-18-laio-plugin-support`) but
never documented: the session-tmux README omits the `{{tmux_socket}}` template variable entirely,
and no sample `laio.yaml` or wiring example exists anywhere in the repo. Users cannot discover
how to configure this without reading internal specs or test code.

## What Changes

- Add `{{tmux_socket}}` to the template-variable table in `plugins/session-tmux/README.md`.
- Add a "Laio integration" section to `plugins/session-tmux/README.md` with two worked examples:
  per-project config (`{{worktree_path}}/.swm/laio.yaml`) and global config (fixed path + `--var`).
- Add a sample `plugins/session-tmux/examples/laio.yaml` showing a realistic multi-window layout
  that works with swm's socket model (using laio's `path:` variable and `force_new_windows`).

## Non-goals

- Changing any runtime behaviour of `pane_group_command` or `{{tmux_socket}}` expansion.
- Adding laio documentation to the root README or the host CLI README.
- Documenting Zellij or other multiplexers.

## Capabilities

### New Capabilities

_(none)_

### Modified Capabilities

_(none — documentation-only change, no spec-level requirement changes)_

## Impact

- `plugins/session-tmux/README.md` — extended with new section and updated template-variable table.
- `plugins/session-tmux/examples/laio.yaml` — new sample file (no build impact).
