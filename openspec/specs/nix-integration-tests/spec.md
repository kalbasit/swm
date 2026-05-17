### Requirement: Integration tests are exercised by `nix flake check`

A dedicated Nix check derivation `checks.swm-integration-tests` SHALL be added so that the
full integration test suite runs whenever `nix flake check` is evaluated. The check MUST
require no network access at test time.

#### Scenario: Integration check appears in flake check output
- **WHEN** a developer runs `nix flake check`
- **THEN** `checks.<system>.swm-integration-tests` is evaluated and the integration test suite passes

#### Scenario: Integration check produces no installed binary
- **WHEN** `nix build .#checks.<system>.swm-integration-tests` is run
- **THEN** the build succeeds and `$out` is an empty directory (the derivation is a test runner, not an application)

### Requirement: Pre-built plugin binaries are injected via environment variables

`TestMain` in `cmd/swm/tests/integration/main_test.go` SHALL check a set of environment
variables before invoking `go build`. When an env var is present and non-empty, the
corresponding binary path is used directly and the `go build` call for that binary is
skipped.

| Binary              | Environment variable          |
|---------------------|-------------------------------|
| vcs-git plugin      | `SWM_PLUGIN_VCS_GIT_BIN`      |
| session-tmux plugin | `SWM_PLUGIN_SESSION_TMUX_BIN` |
| picker-fzf plugin   | `SWM_PLUGIN_PICKER_FZF_BIN`   |
| forge-github plugin | `SWM_PLUGIN_FORGE_GITHUB_BIN` |
| faketmux helper     | `SWM_TEST_FAKETMUX_BIN`       |
| fakefzf helper      | `SWM_TEST_FAKEFZF_BIN`        |

#### Scenario: Env vars bypass go build for plugin binaries
- **WHEN** `SWM_PLUGIN_VCS_GIT_BIN` is set to a valid path
- **THEN** `TestMain` assigns `vcsGitBin` from the env var and does not invoke `go build` for vcs-git

#### Scenario: Env vars bypass go build for test helper binaries
- **WHEN** `SWM_TEST_FAKETMUX_BIN` is set to a valid path
- **THEN** `TestMain` assigns `faketmuxBin` from the env var and does not invoke `go build` for faketmux

### Requirement: Integration tests fall back to `go build` for local development

When the `SWM_PLUGIN_*_BIN` and `SWM_TEST_*_BIN` env vars are absent, `TestMain` MUST
compile plugin and helper binaries exactly as before (via `go build`). Local `go test`
workflows are unaffected.

#### Scenario: No env vars set — go build runs normally
- **WHEN** none of the `SWM_PLUGIN_*_BIN` or `SWM_TEST_*_BIN` env vars are set
- **THEN** `TestMain` compiles all binaries via `go build` and the test suite passes

### Requirement: Test helper packages are available as Nix derivations

Two new Nix packages SHALL expose the `faketmux` and `fakefzf` test helpers so they can
be passed into `checks.swm-integration-tests` as build-time inputs.

- `packages.swm-test-faketmux`: builds `plugins/session-tmux/internal/session/testdata/faketmux`
- `packages.swm-test-fakefzf`: builds `plugins/picker-fzf/internal/picker/testdata/fakefzf`

#### Scenario: Building the faketmux helper
- **WHEN** `nix build .#swm-test-faketmux` is run
- **THEN** the build succeeds and `result/bin/faketmux` is a valid executable

#### Scenario: Building the fakefzf helper
- **WHEN** `nix build .#swm-test-fakefzf` is run
- **THEN** the build succeeds and `result/bin/fakefzf` is a valid executable

### Requirement: Integration check uses only pre-built binaries — no runtime `go build`

The `checks.swm-integration-tests` derivation SHALL set all six `SWM_PLUGIN_*_BIN` and
`SWM_TEST_*_BIN` env vars in `preCheck`, pointing to the outputs of the corresponding
Nix packages. The integration tests MUST complete without network access or a Go workspace.

#### Scenario: Integration check runs without network
- **WHEN** `checks.swm-integration-tests` is evaluated in a Nix sandbox (no network)
- **THEN** all integration tests pass using the pre-built binaries from `self'.packages`
