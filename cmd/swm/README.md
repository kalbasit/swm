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
swm story list
```

Lists all stories and their attached projects.

```sh
swm story remove <name> [-f | --force]
```

Removes a story and all its worktrees. Prompts for confirmation unless `--force` is given.

### `swm workspace`

```sh
swm workspace open [story-name] [--kill-pane]
```

Opens the workspace for a story. Story resolution order:

1. `[story-name]` positional argument (if provided).
2. `$SWM_STORY` environment variable.
3. Default story from config (`default_story`).

If a picker plugin is configured and no story is specified, an interactive list is shown. `--kill-pane` closes the originating tmux pane after switching.

```sh
swm workspace list
```

Lists all active workspaces and their attached projects.

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

# Story used when no story name argument or $SWM_STORY env var is set.
default_story = "_default"

[story]
# Go text/template for the default branch name when `swm story create` is run
# without --branch. The only variable is .Name (the story name). When absent or
# empty, "feat/{{.Name}}" is used, producing "feat/<name>".
#
# Examples:
#   branch_name_template = "feat/{{.Name}}"      # default → feat/my-story
#   branch_name_template = "{{.Name}}"           # bare name → my-story
#   branch_name_template = "users/alice/{{.Name}}" # personal prefix → users/alice/my-story
# branch_name_template = "feat/{{.Name}}"

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
# forge-github: token_path is optional. When absent, the plugin uses
# `gh auth token` (GitHub CLI) or ~/.github_token as fallbacks.
# [plugins.config.forge-github]
# token_path = "~/.config/swm/github_token"

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

| Event                  | Blocking | Working directory | Trigger                                    |
| ---------------------- | -------- | ----------------- | ------------------------------------------ |
| `pre-story-create`     | yes      | `code_root`       | Before a story is created                  |
| `post-story-create`    | no       | `code_root`       | After a story is created                   |
| `pre-story-remove`     | yes      | `code_root`       | Before a story is removed                  |
| `post-story-remove`    | no       | `code_root`       | After a story is removed                   |
| `pre-worktree-create`  | yes      | repo path         | Before a worktree is created for a project |
| `post-worktree-create` | no       | worktree path     | After a worktree is created                |
| `pre-worktree-remove`  | yes      | worktree path     | Before a worktree is removed               |
| `post-worktree-remove` | no       | repo path         | After a worktree is removed                |
| `pre-clone`            | yes      | `code_root`       | Before a repository is cloned              |
| `post-clone`           | no       | repo path         | After a repository is cloned               |
| `pre-workspace-open`   | yes      | `code_root`       | Before a workspace session is opened       |
| `post-workspace-open`  | no       | worktree path     | After a workspace session is opened        |

**Blocking** (`pre-*`): a non-zero exit code aborts the operation immediately.
**Non-blocking** (`post-*`): failures are logged but do not affect the return value.

### Environment variables

Each hook is invoked with the following variables in addition to the calling process's environment:

| Variable            | Description                                                                            |
| ------------------- | -------------------------------------------------------------------------------------- |
| `SWM_HOOK`          | Event name (e.g. `post-worktree-create`)                                               |
| `SWM_STORY`         | Story name                                                                             |
| `SWM_PROJECT_HOST`  | Project host (e.g. `github.com`); empty if no project context                          |
| `SWM_PROJECT_PATH`  | Project path segments joined by `/` (e.g. `kalbasit/swm`); empty if no project context |
| `SWM_WORKTREE_PATH` | Full path to the worktree; empty if not applicable                                     |
| `SWM_REPO_PATH`     | Full path to the canonical repository clone; empty if not applicable                   |

### stdin JSON

Each hook also receives the same data as a JSON object on stdin:

```json
{
  "hook": "post-worktree-create",
  "story": "feat-x",
  "project_host": "github.com",
  "project_path": "kalbasit/swm",
  "worktree_path": "/home/user/code/stories/feat-x/github.com/kalbasit/swm",
  "repo_path": "/home/user/code/repositories/github.com/kalbasit/swm"
}
```

Hooks that do not read stdin will not block — the executor closes the pipe after writing.
