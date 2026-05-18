# swm-plugin-session-tmux

tmux session plugin for swm. Maps swm's workspace/pane-group abstractions onto tmux sockets and sessions.

## Purpose

Implements the `session` capability surface using [tmux](https://github.com/tmux/tmux).

| swm concept | tmux concept                                 |
| ----------- | -------------------------------------------- |
| Workspace   | Dedicated tmux server (one socket per story) |
| Pane group  | tmux session within that server              |

Each story gets its own tmux socket at `$XDG_RUNTIME_DIR/swm/tmux/<story>`, so stories are fully isolated — killing one story's tmux server has no effect on others. `$XDG_RUNTIME_DIR` is set automatically on Linux by systemd/PAM (typically `/run/user/<uid>`). If it is not set in your environment, export it before running swm (e.g. `export XDG_RUNTIME_DIR=/run/user/$(id -u)`).

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

## Configuration

Configure under `[plugins.config.session-tmux]` in `config.toml`:

| Key                  | Type   | Default | Description                                                                                            |
| -------------------- | ------ | ------- | ------------------------------------------------------------------------------------------------------ |
| `pane_group_command` | string | `""`    | Command to run when a pane group (tmux session) is first opened. If empty, tmux opens a default shell. |

### Example

```toml
[plugins]
session = "tmux"

[plugins.config.session-tmux]
# Launch nvim in the first window of every new pane group
pane_group_command = "nvim"
```

### Template variables

When `pane_group_command` is non-empty, these variables are expanded before the command runs:

| Variable            | Expands to                                                                               |
| ------------------- | ---------------------------------------------------------------------------------------- |
| `{{worktree_path}}` | Absolute path to the project's worktree                                                  |
| `{{story_name}}`    | Name of the active story                                                                 |
| `{{project_id}}`    | Project identifier (`<host>/<path>`, e.g. `github.com/kalbasit/swm`)                     |
| `{{tmux_socket}}`   | Absolute path to the story's tmux socket (`$XDG_RUNTIME_DIR/swm/tmux/<story-name>.sock`) |

## Usage

```sh
# Open the workspace for the current story (launches or attaches to tmux)
swm workspace open

# swm sets $SWM_STORY inside the tmux session so hooks and scripts know the active story
echo $SWM_STORY
```

## Socket paths

Tmux sockets are placed at:

```
$XDG_RUNTIME_DIR/swm/tmux/<story-name>.sock
```

You can connect to a story's server directly with:

```sh
tmux -S "$XDG_RUNTIME_DIR/swm/tmux/<story-name>.sock" attach
```

## Laio integration

[laio](https://laio.sh/) is a tmux layout manager that reads a YAML config to
create windows and panes. When used with swm, laio must target swm's per-story socket so its
sessions land on the correct tmux server.

### Per-project config

Place a `laio.yaml` inside the repository (e.g. `.swm/laio.yaml`) and reference it via
`{{worktree_path}}`:

```toml
# config.toml
[plugins.config.session-tmux]
pane_group_command = "laio start --file '{{worktree_path}}/.swm/laio.yaml' --tmux-socket '{{tmux_socket}}' --skip-attach"
```

`path: .` in the laio.yaml resolves relative to the config file, which lives inside the
worktree, so no extra path injection is needed.

### Global config

Keep a single `laio.yaml` at a fixed path and inject the worktree path via laio's Tera
`--var` mechanism:

```yaml
# ~/.config/swm/laio.yaml
path: "{{ path }}" # injected by --var below
windows:
  - name: editor
    panes:
      - shell_command: nvim
  - name: shell
    panes:
      - shell_command: ""
```

```toml
# config.toml
[plugins.config.session-tmux]
pane_group_command = "laio start --file ~/.config/swm/laio.yaml --tmux-socket '{{tmux_socket}}' --skip-attach --var path='{{worktree_path}}'"
```

See [`examples/laio.yaml`](examples/laio.yaml) for a complete annotated sample.

### Why `--skip-attach` is required

laio is invoked inside an already-attached tmux session (the one swm just created and switched
into). Calling `attach-session` or `switch-client` from inside a detached pane would either
fail silently or have no effect. `--skip-attach` tells laio to configure the session without
trying to attach.

## Limitations

- Requires a local tmux installation; remote/SSH-only setups need tmux on the remote host.
- The plugin does not manage tmux configuration (`.tmux.conf`); bring your own config.
- Nested tmux (running swm inside tmux) works but may require `$TMUX` awareness in your config.
