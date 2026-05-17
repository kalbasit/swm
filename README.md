# swm — Story-based Workflow Manager

**swm** organizes your code into per-story git worktrees and launches isolated terminal-multiplexer sessions around them. Each unit of work (a _story_) gets its own branch, its own worktrees across every affected repository, and its own tmux socket — so switching between tasks is instant and context-free.

## Use cases

- Work on multiple features simultaneously without stashing, without branch juggling, and without one project's environment polluting another.
- Clone any repository to its canonical path (`~/code/repositories/<host>/<org>/<repo>`) with a single command and have it immediately available to all stories.
- Open a story's workspace from anywhere and land exactly where you left off.
- Automate pre/post actions (branch protection, issue transitions, Slack notifications) with the hook system.

## Architecture

swm is a **plugin-host CLI**. The host owns the filesystem layout, the story store, plugin lifecycle, and the CLI surface. Every integration is an external plugin binary connected over gRPC.

```
swm (host)
├── session plugin   — terminal multiplexer (bundled: tmux)
├── vcs plugin       — version control (bundled: git)
├── forge plugins    — code-hosting platforms (bundled: github)
├── picker plugin    — interactive selection UI (bundled: fzf)
└── hooks            — plain executables, not gRPC
```

**Five capability surfaces:**

| Capability | What it does                                | Bundled plugin |
| ---------- | ------------------------------------------- | -------------- |
| `session`  | Opens/closes workspaces and pane groups     | `session-tmux` |
| `vcs`      | Clones repos, creates/removes worktrees     | `vcs-git`      |
| `forge`    | Lists and creates pull requests             | `forge-github` |
| `picker`   | Interactive selection prompts               | `picker-fzf`   |
| `hook`     | Lifecycle event scripts (plain executables) | —              |

**Filesystem layout** (defaults):

```
~/code/
├── repositories/          # canonical clones, one per remote
│   └── github.com/org/repo/
└── stories/               # worktrees, one per story per repo
    └── <story>/github.com/org/repo/
```

## Quick start

### Install from source

```sh
go install github.com/kalbasit/swm/cmd/swm@latest
```

The bundled plugins are separate binaries. Build them from the repo:

```sh
git clone https://github.com/kalbasit/swm
cd swm
mkdir -p ~/.local/bin
go build -o ~/.local/bin/swm-plugin-session-tmux ./plugins/session-tmux
go build -o ~/.local/bin/swm-plugin-vcs-git       ./plugins/vcs-git
go build -o ~/.local/bin/swm-plugin-forge-github  ./plugins/forge-github
go build -o ~/.local/bin/swm-plugin-picker-fzf    ./plugins/picker-fzf
```

### Install via Nix

```sh
nix profile install github:kalbasit/swm#swm-full
```

`swm-full` includes the host and all bundled plugins.

### Configure

Create `$XDG_CONFIG_HOME/swm/config.toml` (defaults to `~/.config/swm/config.toml`):

```toml
code_root     = "~/code"
default_story = "_default"

[plugins]
session = "tmux"
vcs     = "git"
picker  = "fzf"
forges  = ["github"]

[plugins.config.forge-github]
token_path = "~/.config/swm/github_token"
```

See [`cmd/swm/README.md`](cmd/swm/README.md) for the full configuration reference.

## Plugin discovery

Plugins are resolved in this order (first match per capability wins):

1. Explicit path in `config.toml` under `[plugins.paths]`
2. `$XDG_DATA_HOME/swm/plugins/<name>/swm-plugin-<capability>-<name>`
3. `swm-plugin-<capability>-<name>` on `$PATH`

## Hook system

Hooks are plain executables placed in tiered directories. All applicable tiers run for each event — they compose rather than override.

| Tier           | Path                                                     |
| -------------- | -------------------------------------------------------- |
| Global         | `$XDG_CONFIG_HOME/swm/hooks/<event>.d/*`                 |
| Per-repository | `<repo>/.swm/hooks/<event>.d/*`                          |
| Per-story      | `$XDG_CONFIG_HOME/swm/stories/<story>/hooks/<event>.d/*` |

`pre-*` hooks abort the operation on non-zero exit. `post-*` hooks log failures but do not abort.

See [`cmd/swm/README.md`](cmd/swm/README.md) for the full list of hook events.

## Module READMEs

| Module                                                   | Description                                 |
| -------------------------------------------------------- | ------------------------------------------- |
| [`cmd/swm`](cmd/swm/README.md)                           | Host CLI — commands, config, plugins, hooks |
| [`sdk/go`](sdk/go/README.md)                             | Go SDK for writing plugins                  |
| [`plugins/session-tmux`](plugins/session-tmux/README.md) | tmux session plugin                         |
| [`plugins/vcs-git`](plugins/vcs-git/README.md)           | git VCS plugin                              |
| [`plugins/forge-github`](plugins/forge-github/README.md) | GitHub forge plugin                         |
| [`plugins/picker-fzf`](plugins/picker-fzf/README.md)     | fzf picker plugin                           |

## Contributing

1. Fork the repo and create a feature branch.
2. All changes require tests (TDD). Run `task test` before pushing.
3. Format and lint: `task fmt && task lint`.
4. Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/).
5. Open a pull request against `main`.

For larger changes, open an issue first to discuss the approach.

## License

Apache 2.0 — see [LICENSE](LICENSE).
