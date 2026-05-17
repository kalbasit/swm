# swm — Host CLI

The swm host binary. It owns the filesystem layout, the story store, plugin lifecycle, and the full CLI surface.

## Commands

### `swm clone <url>`

Clone a repository to its canonical path under `code_root/repositories/`.

```sh
swm clone https://github.com/org/repo
# → ~/code/repositories/github.com/org/repo
```

### `swm story`

Manage stories (units of work).

```sh
swm story create <name> [--branch <branch>]
```

Creates a new story. Defaults the branch to `feat/<name>`.

```sh
swm story remove <name> [-f | --force]
```

Removes a story and all its worktrees. Prompts for confirmation unless `--force` is given.

### `swm workspace`

```sh
swm workspace open
```

Opens the workspace for the current story (as determined by `$SWM_STORY` or the default story). Launches the session plugin.

### `swm pr`

Manage pull requests via the configured forge plugin.

```sh
swm pr list [--story <name>]
```

Lists open pull requests for the current story's projects. Reads `$SWM_STORY` if `--story` is omitted.

```sh
swm pr create --title <title> [--body <text>] [--base <branch>] [--head <branch>] [--draft] [--story <name>]
```

Creates a pull request for the current project. `--base` defaults to `main`; `--head` defaults to the story's branch name.

## Configuration

swm reads `$XDG_CONFIG_HOME/swm/config.toml` (default: `~/.config/swm/config.toml`).

### Full reference

```toml
# Root directory for all code. Repositories land at code_root/repositories/<host>/<org>/<repo>.
code_root = "~/code"

# Story used when no --story flag or $SWM_STORY env var is set.
default_story = "_default"

[plugins]
# Name of the session plugin to load (matches the plugin binary suffix).
session = "tmux"

# Name of the VCS plugin to load.
vcs = "git"

# Name of the picker plugin to load.
picker = "fzf"

# Forge plugins to load. Multiple forges can run simultaneously.
forges = ["github"]

# Optional: override binary paths for specific plugins.
# Key is the full plugin name (capability-name), value is the absolute binary path.
[plugins.paths]
"session-tmux"  = "/usr/local/bin/swm-plugin-session-tmux"
"vcs-git"       = "/usr/local/bin/swm-plugin-vcs-git"
"picker-fzf"    = "/usr/local/bin/swm-plugin-picker-fzf"
"forge-github"  = "/usr/local/bin/swm-plugin-forge-github"

# Per-plugin configuration. Key is the full plugin name.
[plugins.config.forge-github]
token_path = "~/.config/swm/github_token"

[plugins.config.session-tmux]
pane_group_command = ""   # optional custom command run when opening a pane group
```

## Plugin discovery

For each capability, the host resolves the plugin binary in this order (first match wins):

1. **`[plugins.paths]`** — explicit path in config.
2. **XDG data directory** — `$XDG_DATA_HOME/swm/plugins/<name>/swm-plugin-<capability>-<name>`.
3. **`$PATH`** — `swm-plugin-<capability>-<name>`.

Plugin binary naming convention: `swm-plugin-<capability>-<name>`

Examples: `swm-plugin-session-tmux`, `swm-plugin-vcs-git`, `swm-plugin-forge-github`, `swm-plugin-picker-fzf`.

## Hook system

Hooks are plain executables (any language) placed in event-named directories. All tiers run for each event — they compose, not override.

### Tiers (all applicable tiers run per event)

| Tier           | Directory                                               |
| -------------- | ------------------------------------------------------- |
| Global         | `$XDG_CONFIG_HOME/swm/hooks/<event>.d/`                 |
| Per-repository | `<canonical-repo>/.swm/hooks/<event>.d/`                |
| Per-story      | `$XDG_CONFIG_HOME/swm/stories/<story>/hooks/<event>.d/` |

Files within each `.d/` directory run in lexicographic order.

### Hook events

| Event                  | Blocking | Trigger                                    |
| ---------------------- | -------- | ------------------------------------------ |
| `pre-story-create`     | yes      | Before a story is created                  |
| `post-story-create`    | no       | After a story is created                   |
| `pre-story-remove`     | yes      | Before a story is removed                  |
| `post-story-remove`    | no       | After a story is removed                   |
| `pre-worktree-create`  | yes      | Before a worktree is created for a project |
| `post-worktree-create` | no       | After a worktree is created                |
| `pre-worktree-remove`  | yes      | Before a worktree is removed               |
| `post-worktree-remove` | no       | After a worktree is removed                |
| `pre-clone`            | yes      | Before a repository is cloned              |
| `post-clone`           | no       | After a repository is cloned               |
| `pre-workspace-open`   | yes      | Before a workspace session is opened       |
| `post-workspace-open`  | no       | After a workspace session is opened        |

**Blocking** (`pre-*`): a non-zero exit code aborts the operation immediately.
**Non-blocking** (`post-*`): failures are logged but do not affect the return value.

### Environment variables in hooks

Hooks receive the standard swm environment, including `SWM_STORY`, `SWM_CODE_ROOT`, and any variables set by the session plugin.
