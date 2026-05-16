## Why

The swm v2 monorepo ships five Go binaries (the host CLI and four plugins) but has
no Nix packaging.  Users who live in a Nix/NixOS environment cannot install or run any
of them through the flake, and there is no `nix run` entry point.  Adding Nix packages
closes that gap and aligns the project with the ncps/stowix-cli packaging conventions
already used in the author's other projects.

## What Changes

- Add `nix/packages/` with a `flake-module.nix` that is imported by `flake.nix`.
- One sub-directory per binary, each containing a `default.nix` that uses
  `pkgs.buildGoModule`:
  - `nix/packages/swm/` — builds `cmd/swm`
  - `nix/packages/swm-plugin-forge-github/` — builds `plugins/forge-github`
  - `nix/packages/swm-plugin-picker-fzf/` — builds `plugins/picker-fzf`
  - `nix/packages/swm-plugin-session-tmux/` — builds `plugins/session-tmux`
  - `nix/packages/swm-plugin-vcs-git/` — builds `plugins/vcs-git`
- Each package includes a `version.txt` file (initially empty, filled by the release
  process); version falls back to the git revision when the file is empty.
- Each `buildGoModule` source set includes the plugin's own module files **plus** the
  `proto/` and `sdk/go/` local dependencies required by the `replace` directives.
- Add a `swm-full` package defined in `nix/packages/swm-full/default.nix` that uses
  `pkgs.symlinkJoin` to merge all five packages into one store path.
- Expose a flake `apps.default` (and `apps.swm`) entry pointing to the `swm` binary
  inside `swm-full`, via a `nix/apps/flake-module.nix` that is imported by `flake.nix`.
- Wire everything into `flake.nix` by adding the two new imports.

## Capabilities

### New Capabilities

- `packages.swm` — installable Nix package for the host CLI binary.
- `packages.swm-plugin-forge-github` — installable Nix package for the GitHub forge plugin.
- `packages.swm-plugin-picker-fzf` — installable Nix package for the fzf picker plugin.
- `packages.swm-plugin-session-tmux` — installable Nix package for the tmux session plugin.
- `packages.swm-plugin-vcs-git` — installable Nix package for the git VCS plugin.
- `packages.swm-full` — aggregate package containing all binaries via `symlinkJoin`.
- `packages.default` — alias for `swm-full`, making `nix build` work out of the box.
- `apps.swm` / `apps.default` — flake application entry point; `nix run` launches the
  `swm` binary from `swm-full`.

### Modified Capabilities

- `flake.nix` — gains two new imports: `./nix/packages/flake-module.nix` and
  `./nix/apps/flake-module.nix`.

## Impact

- `nix/packages/flake-module.nix` — new file, imports all per-binary sub-modules and
  sets `packages.default = config.packages.swm-full`.
- `nix/packages/swm/default.nix` — new file.
- `nix/packages/swm/version.txt` — new file (empty placeholder).
- `nix/packages/swm-plugin-forge-github/default.nix` — new file.
- `nix/packages/swm-plugin-forge-github/version.txt` — new file.
- `nix/packages/swm-plugin-picker-fzf/default.nix` — new file.
- `nix/packages/swm-plugin-picker-fzf/version.txt` — new file.
- `nix/packages/swm-plugin-session-tmux/default.nix` — new file.
- `nix/packages/swm-plugin-session-tmux/version.txt` — new file.
- `nix/packages/swm-plugin-vcs-git/default.nix` — new file.
- `nix/packages/swm-plugin-vcs-git/version.txt` — new file.
- `nix/packages/swm-full/default.nix` — new file.
- `nix/apps/flake-module.nix` — new file.
- `flake.nix` — add two imports.
- No changes to existing packages, tests, or Go source.
