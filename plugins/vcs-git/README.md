# swm-plugin-vcs-git

git VCS plugin for swm. Handles repository cloning, worktree creation/removal, and branch listing for git repositories.

## Purpose

Implements the `vcs` capability surface using [git](https://git-scm.com). It detects git repositories by the presence of a `.git` directory or file, clones them to the canonical path layout under `code_root/repositories/`, and manages per-story worktrees under `code_root/stories/<story>/`.

## Requirements

- `git` on `$PATH`. Any git version that supports `git worktree` is compatible (2.15+).

```sh
# macOS
brew install git

# Nix
nix profile install nixpkgs#git

# Debian/Ubuntu
apt install git
```

## Configuration

This plugin has no configuration keys. It uses the `git` binary found on `$PATH` and follows standard git configuration (`~/.gitconfig`, `GIT_CONFIG`, etc.).

```toml
[plugins]
vcs = "git"
```

## Usage

The plugin is invoked automatically by swm commands. Direct interaction is through the swm CLI:

```sh
# Clone a repository to its canonical path
swm clone https://github.com/org/repo
# → ~/code/repositories/github.com/org/repo

# Create a story (creates worktrees in all attached projects)
swm story create my-feature

# Remove a story (removes all worktrees)
swm story remove my-feature
```

## Worktree layout

For a story named `my-feature` and a cloned repo at `~/code/repositories/github.com/org/repo`, the worktree is created at:

```
~/code/stories/my-feature/github.com/org/repo/
```

The canonical clone (under `repositories/`) is shared across all stories; worktrees are lightweight references into it.

## Limitations

- Submodules within worktrees are not automatically initialized.
- Shallow clones (`--depth`) are not supported; swm always performs a full clone.
- The plugin uses the system `git` and inherits its credential configuration; set up credential helpers (e.g. `git-credential-manager`) separately.
