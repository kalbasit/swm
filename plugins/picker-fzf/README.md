# swm-plugin-picker-fzf

fzf picker plugin for swm. Provides interactive fuzzy-selection prompts wherever swm needs the user to choose from a list (stories, projects, branches, pull requests).

## Purpose

Implements the `picker` capability surface using [fzf](https://github.com/junegunn/fzf). The host streams candidate items to the plugin over gRPC; the plugin pipes them through `fzf` and returns the selection(s).

## Requirements

- `fzf` must be present on `$PATH`. Install via your system package manager or from the [fzf releases page](https://github.com/junegunn/fzf/releases).

```sh
# macOS
brew install fzf

# Nix
nix profile install nixpkgs#fzf

# Debian/Ubuntu
apt install fzf
```

Any `fzf` version that supports `--read0` and `--print0` is compatible (0.29+).

## Configuration

This plugin has no configuration keys. It uses the `fzf` binary found on `$PATH`.

```toml
[plugins]
picker = "fzf"
```

To use a specific `fzf` binary, set its path explicitly:

```toml
[plugins.paths]
"picker-fzf" = "/usr/local/bin/swm-plugin-picker-fzf"
```

(This sets the swm plugin binary path, not the fzf binary — place your preferred `fzf` earlier on `$PATH`.)

## Usage

The picker is invoked automatically by swm commands that require selection (e.g. `swm workspace open` when no story is active). No direct invocation is needed.

## Limitations

- fzf is launched as a subprocess and inherits the terminal; it will not work in non-interactive (piped) contexts.
- Multi-select support depends on the calling command; not all swm prompts allow multiple selections.
