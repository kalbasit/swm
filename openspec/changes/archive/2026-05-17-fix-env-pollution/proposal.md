## Why

`SWM_HOST_SOCKET`, `SWM_LOG_LEVEL`, and `SWM_PLUGIN_MAGIC_COOKIE` are plugin-internal variables injected into plugin subprocess environments. Because the session plugin launches the terminal multiplexer with `Cmd.Env = nil`, it inherits its own full environment, and these variables flow into every user shell opened in the workspace. There is no spec that defines which variables belong in which execution context, so the correct set is neither enforced nor tested.

## What Changes

- Establish a canonical definition of which `SWM_*` variables belong in each execution context (plugin subprocess, user session, hook process).
- The session plugin strips plugin-internal variables from the environment it hands to the terminal multiplexer, so user shells receive only the user's own environment.

**Environment variable ownership model:**

| Variable | Plugin subprocess | User session | Hook process |
|---|:---:|:---:|:---:|
| `SWM_HOST_SOCKET` | required | must not appear | must not appear |
| `SWM_LOG_LEVEL` | required | must not appear | must not appear |
| `SWM_PLUGIN_MAGIC_COOKIE` | required | must not appear | must not appear |
| `SWM_STORY` | — | present (set by session plugin) | present |
| `SWM_HOOK` | — | — | required |
| `SWM_PROJECT_HOST` | — | — | present |
| `SWM_PROJECT_PATH` | — | — | present |
| `SWM_WORKTREE_PATH` | — | — | present |
| `SWM_REPO_PATH` | — | — | present |

`SWM_STORY` is already propagated into the session (enabling `swm workspace open` to infer the active story without a flag); this change does not alter that behaviour, only makes the requirement explicit.

## Capabilities

### New Capabilities

_(none)_

### Modified Capabilities

- **plugin-lifecycle** — adds requirement: `SWM_*` environment variables MUST be scoped to the context that owns them. Plugin-internal variables MUST NOT appear in user sessions or hook processes.
- **session-tmux** — adds requirement: before launching the multiplexer, the plugin must strip `SWM_HOST_SOCKET`, `SWM_LOG_LEVEL`, and `SWM_PLUGIN_MAGIC_COOKIE` from the inherited environment.

## Non-goals

- Altering the `SWM_STORY` behaviour in sessions (already correct; just made explicit).
- Preventing users from manually setting `SWM_*` vars in their own shell.
- Changes to the plugin gRPC protocol or any proto definitions.

## Impact

- `plugins/session-tmux`: must explicitly filter the environment before launching the multiplexer.
- `openspec/specs/plugin-lifecycle`, `openspec/specs/session-tmux`: specs updated to document env var scoping.
- Capability surfaces affected: **session**, **plugin-lifecycle**.
- No proto changes required.
