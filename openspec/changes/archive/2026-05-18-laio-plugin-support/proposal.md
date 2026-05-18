## Why

When a user sets `pane_group_command = "laio start ..."` in their swm config, laio issues all tmux commands without `-S`/`-L`, so sessions land on the default tmux server instead of swm's per-story socket (`$XDG_RUNTIME_DIR/swm/tmux/<story>.sock`). The result is sessions invisible to swm. Additionally, the existing spec scenario and test fixture reference a non-existent `--config` flag (laio uses `--file`).

## What Changes

- **swm / session-tmux**: Add `{{tmux_socket}}` as a new template variable in `paneGroupCommand`, resolving to the story's socket path (already available as `req.GetWorkspaceId()`).
- **swm / session-tmux**: Fix spec scenario and `tmux_test.go` fixture: `--config` → `--file`, append `--socket {{tmux_socket}}` in the example.
- **laio**: Add `--socket` flag (and honor `LAIO_TMUX_SOCKET` env var) to `laio start`. When set, pass `-S <path>` to every tmux command issued by laio's tmux client.

## Non-goals

- Changing how laio auto-discovers config files when `--file` is omitted.
- Adding a `--config` alias to laio.
- Sandboxing or restricting what commands `pane_group_command` can run.
- Zellij support (laio's secondary multiplexer is out of scope for this integration).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `session-tmux`: The `Custom pane_group_command` scenario gains a new required template variable (`{{tmux_socket}}`), a corrected flag name (`--file` not `--config`), and a new requirement that the socket is injected so pane-group sessions reach the story's tmux server.

## Impact

- **swm** — `plugins/session-tmux/internal/session/tmux.go`: extend `paneGroupCommand` substitution map; update `tmux_test.go` fixture and `openspec/specs/session-tmux/spec.md` scenario.
- **laio** — `src/app/cli/command_line.rs`: add `--socket` arg to `Start`; `src/muxer/tmux/`: thread socket path through to every tmux `Command` builder so `-S <path>` is prepended when set; honor `LAIO_TMUX_SOCKET` env var as fallback.
- No proto changes. No new capability surface. Affects the `session` capability surface only.
