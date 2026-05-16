# swm-plugin-session-tmux

tmux session plugin for swm. Maps swm's workspace/pane-group abstractions onto tmux sockets and sessions.

## Purpose

Implements the `session` capability surface using [tmux](https://github.com/tmux/tmux).

| swm concept | tmux concept                                 |
| ----------- | -------------------------------------------- |
| Workspace   | Dedicated tmux server (one socket per story) |
| Pane group  | tmux session within that server              |

Each story gets its own tmux socket at `$XDG_RUNTIME_DIR/swm/tmux/<story>`, so stories are fully isolated — killing one story's tmux server has no effect on others.

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
$XDG_RUNTIME_DIR/swm/tmux/<story-name>
```

You can connect to a story's server directly with:

```sh
tmux -S "$XDG_RUNTIME_DIR/swm/tmux/<story-name>" attach
```

## Limitations

- Requires a local tmux installation; remote/SSH-only setups need tmux on the remote host.
- The plugin does not manage tmux configuration (`.tmux.conf`); bring your own config.
- Nested tmux (running swm inside tmux) works but may require `$TMUX` awareness in your config.
