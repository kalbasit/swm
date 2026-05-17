## MODIFIED Requirements

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
  from the source fileset because those tests are exercised separately by
  `checks.swm-integration-tests` (see `nix-integration-tests` spec).
- `packages.swm-test-faketmux`: no extra tools; builds only the `faketmux` subpackage.
- `packages.swm-test-fakefzf`: no extra tools; builds only the `fakefzf` subpackage.

#### Scenario: vcs-git tests pass with git available
- **WHEN** `nix build .#swm-plugin-vcs-git` is run
- **THEN** the check phase runs `go test ./...` inside the `plugins/vcs-git` module with `git` on `PATH`, and all tests pass

#### Scenario: session-tmux tests pass without real tmux
- **WHEN** `nix build .#swm-plugin-session-tmux` is run
- **THEN** the check phase runs `go test ./...` inside the `plugins/session-tmux` module (using the in-tree `faketmux` binary built by the test suite), and all tests pass

#### Scenario: forge-github tests pass without network
- **WHEN** `nix build .#swm-plugin-forge-github` is run
- **THEN** the check phase runs `go test ./...` with an in-process `net/http/httptest` mock server, and all tests pass

#### Scenario: swm unit tests pass
- **WHEN** `nix build .#swm` is run
- **THEN** the check phase runs tests under `cmd/swm/internal/` and passes; `tests/integration/` is excluded from the `packages.swm` source fileset and exercised separately by `checks.swm-integration-tests`
