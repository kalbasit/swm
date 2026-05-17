# Proposal: nix-integration-tests

## Why

`cmd/swm/tests/integration/` is excluded from the `packages.swm` Nix derivation because it
calls `go build` on all plugins at runtime using the Go workspace, which is incompatible with
`buildGoModule`'s single-module sandbox. As a result, `nix flake check` (the sole CI gate)
never runs these tests, leaving an entire layer of end-to-end coverage invisible to CI.

## What Changes

- Add `checks.swm-integration-tests`: a new `buildGoModule` derivation that includes only
  `cmd/swm/tests/integration/` plus all required sources, with pre-built plugin binaries
  injected via environment variables so no `go build` step is needed at test time.
- Modify `cmd/swm/tests/integration/main_test.go`: when `SWM_PLUGIN_<NAME>_BIN` env vars
  are set, skip the `go build` step and use those paths directly; fall back to building for
  local development.
- Update `openspec/specs/nix-packages/spec.md`: replace the "integration tests are absent"
  scenario with a "integration tests run via a dedicated check" requirement.

## Capabilities

### New Capabilities

- `nix-integration-tests` — a `nix flake check` entry (`checks.swm-integration-tests`) that
  runs the full integration suite by consuming the pre-built plugin package outputs as binary
  inputs.

### Modified Capabilities

- `nix-packages` — the existing requirement "Packages run tests in the Nix check phase"
  documents the exclusion of `cmd/swm/tests/integration/` from `packages.swm`. That scenario
  changes: integration tests are no longer absent; they run under a sibling check derivation.

## Impact

- `nix/checks/flake-module.nix`: gains `checks.swm-integration-tests`.
- `nix/packages/swm/default.nix`: no change (integration source stays excluded from
  `packages.swm`; the new check derivation is its own derivation).
- `cmd/swm/tests/integration/main_test.go`: environment-variable injection of binary paths.
- `openspec/specs/nix-packages/spec.md`: updated scenario.
- No proto changes. No public API changes.
- Affected capability surfaces: none (this is a build/test infrastructure change).
