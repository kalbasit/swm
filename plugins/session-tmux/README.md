# swm-plugin-session-tmux

tmux session plugin for swm. Maps swm's workspace/pane-group abstractions onto tmux sockets and sessions.

## Purpose

Implements the `session` capability surface using [tmux](https://github.com/tmux/tmux).

| swm concept | tmux concept                                 |
| ----------- | -------------------------------------------- |
| Workspace   | Dedicated tmux server (one socket per story) |
| Pane group  | tmux session within that server              |

Each story gets its own tmux socket at `$XDG_RUNTIME_DIR/swm/tmux/<story>.sock`, so stories are fully isolated — killing one story's tmux server has no effect on others. `$XDG_RUNTIME_DIR` is set automatically on Linux by systemd/PAM (typically `/run/user/<uid>`). If it is not set in your environment, export it before running swm (e.g. `export XDG_RUNTIME_DIR=/run/user/$(id -u)`).

## Requirements

- `tmux` 3.2 or later on `$PATH`.

```sh
# macOS
brew install tmux

# Nix
nix profile install nixpkgs#tmux

# Debian/Ubuntu
apt install tmux
```

## Layout configuration

When a pane group (tmux session) is first opened, the plugin looks for a layout
config file in two locations, preferring the per-repo one:

| Priority | Path                                     | Scope    |
| -------- | ---------------------------------------- | -------- |
| 1        | `<worktree>/.swm/session-tmux.toml`      | per-repo |
| 2        | `$XDG_CONFIG_HOME/swm/session-tmux.toml` | global   |

If neither file is found, a single default shell pane is opened.

### Template variables

Config values are Go [`text/template`](https://pkg.go.dev/text/template) strings.
The following variables are available anywhere in the file:

| Variable            | Expands to                                                                               |
| ------------------- | ---------------------------------------------------------------------------------------- |
| `{{.WorktreePath}}` | Absolute path to the project's worktree                                                  |
| `{{.StoryName}}`    | Name of the active story                                                                 |
| `{{.TmuxSocket}}`   | Absolute path to the story's tmux socket (`$XDG_RUNTIME_DIR/swm/tmux/<story-name>.sock`) |

### Schema reference

```toml
# Top-level options
path           = "{{.WorktreePath}}"  # base path for all windows/panes (Go template)
shell          = "/bin/zsh"           # shell binary; inherits $SHELL when unset
pane_cmd_delay = 50                   # ms to wait between send-keys calls (default: 0)

[env]
# Session-level environment variables set before layout is applied.
EDITOR = "nvim"

[[startup]]
# Commands sent to the initial pane before any windows/panes are created.
# Useful for bootstrapping tools (e.g. mise, nix develop).
command = "mise install"
# args   = []  # optional argument list; each arg is shell-quoted when non-trivial

[[windows]]
name           = "editor"    # required; must be non-empty
path           = "src"       # optional; resolved relative to top-level path
flex_direction = "column"    # "column" (vertical stacked, default) | "row" (side by side)

  [windows.env]
  NODE_ENV = "development"

  [[windows.panes]]
  flex           = 2           # relative size weight (default: 1, minimum: 1)
  flex_direction = "row"       # overrides the window split direction for children
  path           = "frontend"  # resolved relative to window path
  focus          = true        # focus this pane after layout (max one per window)
  zoom           = false       # zoom this pane (applied after focus; max one per window)
  commands       = ["nvim ."]  # sent via tmux send-keys

    # Panes are recursive — a pane with child panes becomes a split container.
    [[windows.panes.panes]]
    flex     = 1
    commands = ["npm test -- --watch"]

  [[windows.panes]]
  flex     = 1
  commands = ["bash"]
```

**Constraints:**

- At least one `[[windows]]` is required.
- `flex` must be ≥ 1.
- At most one pane per window may have `focus = true`.
- At most one pane per window may have `zoom = true`.

### Flex layout

Sibling panes are sized proportionally by their `flex` weights:

- Two panes with `flex = 1` each → 50 / 50 split.
- Panes with `flex = 2` and `flex = 1` → 66 / 33 split.

`flex_direction = "column"` (default) stacks panes top/bottom; `"row"` places them side by side. Each pane container can override the direction independently, enabling arbitrarily nested grid layouts.

### Examples

**Global config — open nvim and a shell for every project:**

```toml
# ~/.config/swm/session-tmux.toml
path = "{{.WorktreePath}}"

[[windows]]
name = "code"

  [[windows.panes]]
  flex     = 2
  commands = ["nvim ."]
  focus    = true

  [[windows.panes]]
  flex = 1

[[windows]]
name = "shell"
```

**Per-repo config — editor with tests and git log beside it:**

```toml
# <repo>/.swm/session-tmux.toml
path = "{{.WorktreePath}}"

[[windows]]
name           = "editor"
flex_direction = "row"

  [[windows.panes]]
  flex     = 3
  commands = ["nvim ."]
  focus    = true

  [[windows.panes]]
  flex           = 1
  flex_direction = "column"

    [[windows.panes.panes]]
    commands = ["go test ./..."]

    [[windows.panes.panes]]
    commands = ["git log --oneline"]

[[windows]]
name = "shell"
```

**Bootstrap with mise before opening the layout:**

```toml
# <repo>/.swm/session-tmux.toml
path = "{{.WorktreePath}}"

[[startup]]
command = "mise install"

[[windows]]
name = "code"

  [[windows.panes]]
  commands = ["nvim ."]
```

## Pane primitives

Beyond workspace and pane-group lifecycle, the plugin implements the `Session`
service's four pane-level RPCs. They exist so that a program which needs to run
and drive something in a pane can do it through swm rather than shelling out to
`tmux` — swm owns multiplexer access, and a direct `tmux` call would not survive
a switch to another provider.

| RPC         | tmux equivalent                 | Notes                                                         |
| ----------- | ------------------------------- | ------------------------------------------------------------- |
| `OpenPane`  | `new-window -P -F '#{pane_id}'` | `argv` is shell-quoted; empty `argv` starts the default shell |
| `ListPanes` | `list-panes -a -F …`            | Every live workspace when `workspace_id` is empty             |
| `SendText`  | `send-keys -l -- …`             | Refuses a focused pane unless `allow_focused` is set          |
| `ClosePane` | `kill-pane`                     | Succeeds when the pane is already gone                        |

`pane_id` is an opaque handle (`%4` on tmux). Pass it back verbatim; do not
parse it. It is meaningful only within the workspace that produced it, which is
why every request carrying one also carries `workspace_id`.

### Sending text into a pane someone is using

Text delivered by `SendText` arrives exactly as if it had been typed, so
whatever the pane is currently displaying consumes it. If a person is sitting at
a `y/N` confirmation, injected text answers it on their behalf.

The plugin therefore **refuses to deliver into a focused pane** — one that is
the active pane, of the active window, of a session with an attached client, so
that a human's keystrokes would currently land there. The call fails with
`FAILED_PRECONDITION`; set `allow_focused` to send anyway.

`ListPanes` reports the same `focused` flag from the same query, so a caller can
see in advance which panes will be refused. The check is racy by construction —
someone can focus a pane the instant after it passes — so it is a guard against
the common accident, not a guarantee. A caller that sets `allow_focused` owns
the consequences.

`delay_ms` waits before delivering, the same mitigation `pane_cmd_delay` offers
for layout commands: a program that has just started may not have installed its
input handler yet. To leave a gap between the text and Enter, send them as two
calls — an empty `text` with `submit = true` delivers only the submit key.

## Plugin configuration (`config.toml`)

The plugin itself needs no configuration for layout — the `session-tmux.toml` files (above) are sufficient. The following options can be set under `[plugins.config.session-tmux]` when needed:

| Key                  | Type   | Default | Description                                                                                                                                                                                                       |
| -------------------- | ------ | ------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `pane_group_command` | string | `""`    | Shell command run when a pane group is first opened. Takes precedence over layout config when set. Supports `{{.WorktreePath}}`, `{{.StoryName}}`, `{{.ProjectID}}`, and `{{.TmuxSocket}}` Go template variables. |

## Usage

```sh
# Open the workspace for the current story (launches or attaches to tmux)
swm workspace open

# swm sets $SWM_STORY inside the tmux session so hooks and scripts know the active story
echo $SWM_STORY
```

## Socket paths

Tmux sockets are placed at:

```text
$XDG_RUNTIME_DIR/swm/tmux/<story-name>.sock
```

You can connect to a story's server directly with:

```sh
tmux -S "$XDG_RUNTIME_DIR/swm/tmux/<story-name>.sock" attach
```

## Limitations

- Requires a local tmux installation; remote/SSH-only setups need tmux on the remote host.
- The plugin does not manage tmux configuration (`.tmux.conf`); bring your own config.
- Nested tmux (running swm inside tmux) works but may require `$TMUX` awareness in your config.
