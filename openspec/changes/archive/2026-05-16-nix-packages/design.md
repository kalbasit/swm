## Context

The swm v2 monorepo contains five Go binaries, each in its own Go module:

| Binary | Module path |
|--------|-------------|
| `swm` | `cmd/swm` |
| `swm-plugin-forge-github` | `plugins/forge-github` |
| `swm-plugin-picker-fzf` | `plugins/picker-fzf` |
| `swm-plugin-session-tmux` | `plugins/session-tmux` |
| `swm-plugin-vcs-git` | `plugins/vcs-git` |

Every plugin module has `replace` directives pointing at the local `proto/` and `sdk/go/`
modules.  The Go workspace (`go.work`) knits them together for development, but each
module is vendored independently for Nix builds.

Currently `flake.nix` has no `packages` or `apps` outputs — there is no way to install
or run any swm binary via Nix.  The repository already follows the flake-parts
`perSystem` pattern for devshells, formatters, checks, and pre-commit hooks.

Reference implementations for the build pattern:
- `github.com/kalbasit/ncps` — single Go module, `buildGoModule`, `wrapProgram`, `version.txt`
- `Stowix/stowix-cli` — single Go module, `buildGoModule`, `version.txt`, packages as a flake-parts sub-module

## Goals / Non-Goals

**Goals:**
- Package each binary as a standalone Nix derivation using `buildGoModule`.
- Provide a `swm-full` aggregate package via `pkgs.symlinkJoin`.
- Expose `apps.default` / `apps.swm` so `nix run` works.
- Follow the exact same flake-parts / file-layout conventions as ncps and stowix-cli.
- Keep each package definition in its own sub-directory under `nix/packages/`.

**Non-Goals:**
- Docker images (not needed for a CLI + plugins).
- Cross-compilation or multi-arch matrix beyond what the flake's `systems` list already covers.
- Release automation or version tagging (out of scope for this change).

## Decisions

### One `buildGoModule` per Go module

Each of the five modules is vendored separately (`go.sum` is per-module) and has its own
`replace` directives.  A single unified `buildGoModule` call for the whole workspace
would require a single flat vendor tree, which the Go toolchain does not support for
workspaces.  Keeping one derivation per module mirrors how the Go toolchain itself treats
them and gives independent `vendorHash` values.

**Alternatives considered:**
- A single workspace-level build: not supported by `buildGoModule` (`go work vendor`
  produces a merged tree that `nix build` cannot hash reproducibly without careful
  massaging).

### Source filesets include `proto/` and `sdk/go/`

Every plugin (and `cmd/swm`) uses `replace` directives for the local `proto` and `sdk/go`
modules.  Nixpkgs' `buildGoModule` resolves replace directives against `src`, so the
source fileset for each package must include those two directories in addition to the
module's own sources.  The `root` of the fileset is always the repository root (`../../..`
relative to the package's `default.nix`).

### `swm-full` uses `pkgs.symlinkJoin`

`symlinkJoin` is the idiomatic Nix way to merge several derivations into one store path.
It is used by nixpkgs itself for aggregate packages.  The result is a single path that
contains all binaries, which is what a flake `apps` entry needs.

### `apps` module is separate from `packages`

The flake apps output is a small concern (one or two attribute lines) and is logically
distinct from package definitions.  A dedicated `nix/apps/flake-module.nix` keeps
`nix/packages/` focused on building and makes it easy to add more applications later
without touching package definitions.

### `version.txt` fallback pattern

Following ncps and stowix-cli, each package directory contains an empty `version.txt`.
When the file is empty the derivation falls back to `self.rev or self.dirtyRev`.  This
allows release tooling to stamp a semver tag without changing the Nix expression itself.

### Tests run in the Nix check phase with `doCheck = true`

All five packages enable the check phase.  No package requires network access; each
test suite's external dependencies are provided via `nativeBuildInputs`:

| Package | Test strategy | Extra `nativeBuildInputs` |
|---------|--------------|--------------------------|
| `swm-plugin-vcs-git` | calls `git` via os/exec | `pkgs.git` |
| `swm-plugin-session-tmux` | builds `faketmux` with `go build` at test time | _(none, Go compiler already present)_ |
| `swm-plugin-picker-fzf` | builds `fakefzf` with `go build` at test time | _(none)_ |
| `swm-plugin-forge-github` | pure in-process `net/http/httptest` mock | _(none)_ |
| `swm` (cmd/swm unit tests) | unit tests only; integration tests (which compile all plugins at runtime) are excluded from the fileset | `pkgs.git` |

For `cmd/swm`, the `tests/integration/` directory is **excluded from the fileset**.
Those tests compile all plugin modules at runtime via `go build` using the workspace,
which cannot be done inside a single-module `buildGoModule` sandbox.  Unit tests under
`cmd/swm/internal/` are sandboxed correctly and exercise the core logic.

## Risks / Trade-offs

- **Five separate `vendorHash` values** — each must be updated when any dependency of
  that module changes.  This is a maintenance cost inherent to having five modules.
  Mitigation: keep `vendorHash = "sha256-..."` in the per-package `default.nix`; a
  helper script or CI can regenerate them.

- **`replace` directives + filesets** — if a new local dependency is added to a module's
  `go.mod` `replace` block, its source tree must also be added to that package's fileset.
  Mitigation: document this in the package's `default.nix` comment.

- **`cmd/swm` integration tests excluded from Nix** — integration tests that compile
  plugins at runtime require the full Go workspace, which is incompatible with single-
  module `buildGoModule`.  Mitigation: those tests continue to run via `task test` in the
  devshell, which has the full workspace.  The Nix check phase covers all unit tests.
