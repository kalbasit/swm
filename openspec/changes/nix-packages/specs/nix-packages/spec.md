## ADDED Requirements

### Requirement: Per-binary Nix packages exist
The flake SHALL expose one installable Nix package per swm binary under `packages.<name>`:
- `packages.swm` builds `cmd/swm`
- `packages.swm-plugin-forge-github` builds `plugins/forge-github`
- `packages.swm-plugin-picker-fzf` builds `plugins/picker-fzf`
- `packages.swm-plugin-session-tmux` builds `plugins/session-tmux`
- `packages.swm-plugin-vcs-git` builds `plugins/vcs-git`

Each package MUST be a `pkgs.buildGoModule` derivation whose source fileset includes the
binary's own module directory (`cmd/<name>` or `plugins/<name>`), plus the shared local
modules `proto/` and `sdk/go/` required by `replace` directives.

#### Scenario: Building the swm host binary
- **WHEN** a user runs `nix build .#swm`
- **THEN** the build succeeds and `result/bin/swm` is a working executable

#### Scenario: Building a plugin binary
- **WHEN** a user runs `nix build .#swm-plugin-session-tmux`
- **THEN** the build succeeds and `result/bin/swm-plugin-session-tmux` is a working executable

#### Scenario: Replace directives resolve inside the build
- **WHEN** a plugin package is built by Nix
- **THEN** the `proto/` and `sdk/go/` source trees are included in the sandbox so `replace` directives resolve correctly

### Requirement: Version falls back to git revision
Each package SHALL determine its version as follows:
1. Read `version.txt` from the package's `nix/packages/<name>/` directory.
2. If the file is non-empty (after trimming whitespace), use that value.
3. Otherwise use `self.rev or self.dirtyRev` from the flake self reference.

#### Scenario: Clean release build
- **WHEN** `version.txt` contains `v1.2.3`
- **THEN** the built binary reports version `v1.2.3`

#### Scenario: Development build without a version tag
- **WHEN** `version.txt` is empty
- **THEN** the derivation uses the git commit SHA as the version

### Requirement: Packages run tests in the Nix check phase
Each `buildGoModule` derivation SHALL set `doCheck = true`.  No package requires
network access.  Each package MUST provide the external tools its test suite needs via
`nativeBuildInputs`:

- `packages.swm-plugin-vcs-git`: MUST include `pkgs.git` (tests invoke `git` via `os/exec`).
- `packages.swm-plugin-session-tmux`: no extra tools (tests build `faketmux` from
  testdata using the Go compiler already present in the build environment).
- `packages.swm-plugin-picker-fzf`: no extra tools (tests build `fakefzf` from testdata
  using the Go compiler).
- `packages.swm-plugin-forge-github`: no extra tools (tests use `net/http/httptest`
  in-process mock, no external service).
- `packages.swm`: MUST include `pkgs.git`; MUST exclude `cmd/swm/tests/integration/`
  from the source fileset because those tests compile all plugin modules at runtime via
  `go build` with the workspace, which is incompatible with a single-module sandbox.

#### Scenario: vcs-git tests pass with git available
- **WHEN** `nix build .#swm-plugin-vcs-git` is run
- **THEN** the check phase runs the test suite against a real `git` binary and all tests pass

#### Scenario: session-tmux tests pass without real tmux
- **WHEN** `nix build .#swm-plugin-session-tmux` is run
- **THEN** the check phase compiles `faketmux` from testdata and runs tests against it, with no real tmux binary required

#### Scenario: forge-github tests pass without network
- **WHEN** `nix build .#swm-plugin-forge-github` is run in a network-restricted sandbox
- **THEN** the check phase runs all tests using the in-process httptest mock and passes

#### Scenario: swm unit tests pass
- **WHEN** `nix build .#swm` is run
- **THEN** the check phase runs tests under `cmd/swm/internal/` and passes; the `tests/integration/` package is absent from the source tree and is not compiled
