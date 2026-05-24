## Why

The swm monorepo maintains 7 Go modules and a Nix flake with 5 external inputs, all updated manually today — security patches and upstream releases accumulate unnoticed. Integrating Renovate automates dependency PR creation with controlled grouping and scheduling, reducing toil and improving supply-chain hygiene.

## What Changes

- Add `renovate.json` at the repository root with Mend Renovate configuration.
- Configure Go module update rules across all 7 `go.mod` files (one PR per module, grouped minor/patch).
- Configure Nix flake input update rules (one PR per input, weekly schedule).
- Configure GitHub Actions dependency updates (one PR per action, grouped minor/patch).
- Apply standard PR labels (`dependencies`, `renovate`) and assign reviewers.

## Capabilities

### New Capabilities

- `renovate-config` — Specifies the Renovate configuration contract: which package ecosystems are covered, update grouping strategy, scheduling rules, PR labels, and reviewer assignment.

### Modified Capabilities

_(none — no existing spec-level behavior changes)_

## Non-goals

- Modifying the swm plugin protocol or any gRPC surface.
- Automated PR merging (Renovate opens PRs; humans review and merge).
- Managing runtime dependencies of swm itself (Renovate is a repo tooling concern only).
- Pinning all dependencies to exact versions (semver ranges remain in use per current convention).

## Impact

- **Affected files**: `renovate.json` (new), optionally `.github/renovate.json5` as alias location.
- **Capability surfaces**: none — this is repository infrastructure, not a swm plugin capability.
- **Proto changes**: none.
- **CI**: Renovate PRs will trigger the existing CI workflow unchanged; no CI modifications needed.
- **Dependencies**: no new runtime or build dependencies added to any Go module or Nix package.
