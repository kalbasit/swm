## Context

When the host launches a plugin subprocess via go-plugin, it prepends `SWM_HOST_SOCKET` and `SWM_LOG_LEVEL` to the command's `Env` slice; go-plugin then appends the full `os.Environ()` and injects `SWM_PLUGIN_MAGIC_COOKIE` itself. The session-tmux plugin subsequently launches the tmux server with `exec.Cmd.Env = nil`, so tmux inherits the plugin's complete environment — including all three plugin-internal variables. Every shell opened in any tmux window then sees them.

The fix is a single, self-contained change in the session plugin: explicitly construct the tmux command's `Env` instead of relying on `nil` inheritance, omitting the three plugin-internal variables.

## Goals / Non-Goals

**Goals:**
- Strip `SWM_HOST_SOCKET`, `SWM_LOG_LEVEL`, and `SWM_PLUGIN_MAGIC_COOKIE` from the environment passed to the tmux server at launch time.
- Define and codify the canonical ownership model for `SWM_*` variables across all execution contexts.

**Non-Goals:**
- Changing how `SWM_STORY` is propagated — the proposal marks this behaviour as already correct and explicitly out of scope.
- Modifying the plugin-manager launch path or go-plugin configuration.
- Scrubbing any variables from hook processes — hooks intentionally receive a rich context.

## Decisions

### Decision 1: Fix in the session plugin, not in the host

**Chosen:** The session-tmux plugin filters its inherited environment before constructing the tmux `exec.Cmd`.

**Alternatives considered:**

- *Set `SkipHostEnv: true` in pluginmgr* — would suppress all environment inheritance for every plugin, requiring explicit plumbing of the full user environment through the manager. Changes every plugin's launch path unnecessarily.
- *Host-side strip list embedded in the launch env* — unnecessary indirection; the session plugin is the only consumer that needs to present a clean environment to a user-facing process.

### Decision 2: Denylist, not allowlist

Filter by removing the three known plugin-internal variables rather than building the env from an explicit allowlist of safe vars.

**Rationale:** An allowlist would require enumerating every variable a user might legitimately pass to their session, which is unbounded and fragile across user environments. A denylist is surgical and stable — the set of plugin-internal variables is small and fully known at compile time. The three variables to strip are: `SWM_HOST_SOCKET`, `SWM_LOG_LEVEL`, `SWM_PLUGIN_MAGIC_COOKIE`.

## Risks / Trade-offs

- [Future plugin-internal vars] If a new `SWM_*` variable is added for plugin communication, it must also be added to the denylist in the session plugin. → Mitigation: the spec requirement and its integration test make omissions visible at test time.
- [SWM_LOG_LEVEL as a user var] A user who has exported `SWM_LOG_LEVEL` in their own shell before launching swm will not see it in the tmux session. → Acceptable: the name clearly signals swm-internal use; no legitimate user workflow depends on it in the session.

## Migration Plan

No migration required. The change is contained entirely to the session plugin and is transparent to users — their shell environment is cleaner after the fix, with no other behavioural change.
