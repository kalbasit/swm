## Context

swm's `session-tmux` plugin creates per-story tmux servers at `$XDG_RUNTIME_DIR/swm/tmux/<story>.sock`. When a user configures `pane_group_command`, swm runs that command as the initial shell process of the pane-group's tmux session (via `tmux -S <sock> new-session -d -s <name> -c <path> COMMAND`). laio currently issues all tmux commands without `-S`, so any sessions or windows it creates land on the default tmux server rather than swm's story socket — they are invisible to swm.

An additional correctness bug: the existing spec scenario and test fixture reference `--config`, a flag that does not exist in laio (the correct flag is `--file`).

## Goals / Non-Goals

**Goals:**
- laio sessions and windows land on the swm-managed per-story socket.
- Users can write `pane_group_command = "laio start --file {{worktree_path}}/.swm/laio.yaml --socket {{tmux_socket}} --skip-attach"` and have it work end-to-end.
- Correct the broken `--config` flag in the spec and test fixture.

**Non-Goals:**
- Zellij socket support in laio.
- Changing laio's default config-discovery when `--file` is omitted.
- Modifying how swm names pane groups.

## Decisions

### D1: laio interface — `--socket` flag + `LAIO_TMUX_SOCKET` env var, flag takes priority

A `--socket` flag on `laio start` accepts an absolute path to an external tmux socket. If absent, laio falls back to `LAIO_TMUX_SOCKET`. When either is set, laio prepends `-S <path>` to every tmux command it issues.

**Why over flag-only**: An env var lets wrapper scripts and hooks call laio without threading the socket argument through command templates explicitly. **Why over env-only**: An explicit flag is unambiguous when scripting and is self-documenting in `pane_group_command`.

### D2: laio in-session mode — configure current session when already inside tmux

When `--socket` is provided (or `LAIO_TMUX_SOCKET` set) and laio detects it is running inside a tmux pane (`is_inside_session()` = true), it skips `new-session` and instead configures the current session's windows and panes. All configured windows are created via `new-window` (`force_new_windows = true`); after all commands are flushed, the original window (the one laio was launched in) is killed so focus lands on the correct configured window. This preserves the session name that swm created (`github•com/kalbasit/swm`) so swm's `SwitchTo` continues to work.

**Why not create a new session**: swm records and navigates by the session name it created. A second session with a laio-chosen name would be unreachable via `swm workspace open`.

**Why `is_inside_session()` rather than a new flag**: laio already has this check for interactive use; reusing it avoids bespoke swm-awareness in laio's interface.

**Why `force_new_windows=true` + kill original**: The original window (where laio runs) disappears when laio exits. Without `force_new_windows`, the first configured window maps to the existing original window and closes with laio. By creating every configured window via `new-window` and then killing the original explicitly, all configured windows outlive the laio process.

### D3: swm template variable — `{{tmux_socket}}`

Add `{{tmux_socket}}` → `req.GetWorkspaceId()` to `paneGroupCommand`'s substitution map alongside the existing `{{worktree_path}}`, `{{story_name}}`, and `{{project_id}}` variables. The value is the absolute path of the story's socket, already present in the request.

**Why a template variable over auto-injecting an env var**: template variables are visible in config, auditable, and consistent with the established pattern. Auto-injection would make the env var a hidden side effect.

### D4: Common examples

**Per-project config** (`laio.yaml` lives inside the repo, `path: .` resolves to the
project directory because the config and the project share the same directory):

```toml
[plugins.session-tmux]
pane_group_command = "laio start --file {{worktree_path}}/.swm/laio.yaml --socket {{tmux_socket}} --skip-attach"
```

**Global config** (`laio.yaml` lives at a fixed location such as `~/.config/swm/laio.yaml`;
`path: .` would resolve to the config file's parent, so the worktree path must be passed
explicitly via laio's Tera `--var` mechanism):

```yaml
# ~/.config/swm/laio.yaml
path: "{{ path }}"   # injected by --var path=<worktree_path> below
```

```toml
[plugins.session-tmux]
pane_group_command = "laio start --file ~/.config/swm/laio.yaml --socket {{tmux_socket}} --skip-attach --var path={{worktree_path}}"
```

`--skip-attach` is required: laio is running inside the new session already; calling `attach-session` or `switch-client` from within a detached pane would fail or have no effect.

### D6: Bootstrap session in OpenWorkspace — lazy project session creation

`OpenWorkspace` creates a single bootstrap session named after the story (e.g., `feat-x`) to keep the tmux server alive; project sessions (e.g., `github•com/kalbasit/swm`) are created lazily by `OpenPaneGroup`.

**Why not pre-create project sessions**: The original design pre-created one session per worktree in `OpenWorkspace`. This raced with `pane_group_command`: by the time `OpenPaneGroup` ran, the session already existed, so `new-session` was skipped and `pane_group_command` was never invoked as the initial process of the session.

**Why the story name for the bootstrap session**: Project sessions use `host/org/repo` format (e.g., `github•com/kalbasit/swm`), which always contains `/` and `•`. The story name is short and never matches this pattern, so the two namespaces never collide.

### D5: Spec and test fixture correction

Both `openspec/specs/session-tmux/spec.md` (Scenario: Custom pane_group_command) and `tmux_test.go` (TestOpenPaneGroup_WithPaneGroupCommand) use `--config`, which does not exist in laio. Both are updated to `--file` and the example is extended to include `--socket {{tmux_socket}}`.

## Risks / Trade-offs

- **`$TMUX` availability in detached panes**: tmux sets `$TMUX` in every pane's environment including detached sessions; laio's `is_inside_session()` relies on this. If the env var is absent in a particular execution context, laio falls back to creating a new session on the socket (sessions land on the right server, but swm's navigation by session name will not work). Mitigation: add an integration test in laio that simulates the detached pane scenario.
- **Window lifecycle in in-session mode**: All configured windows are created via `new-window` (`force_new_windows = true`) and the original laio-started window is killed after flush. This differs from the normal flow where the first configured window is the session's initial window; here it is created fresh like all others.
- **Socket path baked into config**: `{{tmux_socket}}` is expanded at `OpenPaneGroup` call time, so configs remain correct across `$XDG_RUNTIME_DIR` changes. No action needed.
