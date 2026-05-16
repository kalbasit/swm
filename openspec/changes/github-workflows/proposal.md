# Proposal: Adopt GitHub Workflows from ncps

## Why

`swm` has no CI/CD automation: no linting, no test runs on PRs, no release automation,
no security scanning, and no Nix flake maintenance. Adopting the proven workflow set
from `ncps` closes this gap with minimal adaptation effort.

## What Changes

- **New**: `.github/workflows/ci.yml` — run `nix flake check` (fmt, lint, test via `task`)
  on PRs and pushes to `main`/`release-*`; auto-update Nix vendor hash when `go.mod`/`go.sum`
  change (non-fork PRs only)
- **New**: `.github/workflows/releases.yml` — on `v*.*.*` tags, run `nix flake check`,
  create a GitHub Release (draft, with pre-release detection for alpha/beta/rc)
- **New**: `.github/workflows/semantic-pull-request.yml` — enforce Conventional Commits
  format on all PR titles targeting `main`/`release-*`
- **New**: `.github/workflows/devskim.yml` — weekly + on-push/PR security scan via
  Microsoft DevSkim, upload SARIF to GitHub Security tab
- **New**: `.github/workflows/flake-update.yml` — weekly automated `nix flake update`
  + `go mod tidy`, opened as a PR with auto-merge enabled
- **New**: `.github/workflows/backport.yml` — bot-driven backports to `release-*` branches
  triggered by label or merge event
- **Skip**: `build.yml` / `releases.yml` Docker+Helm sections — `swm` is a CLI tool,
  no container images or Helm charts
- **Skip**: `fuzz.yml` — no fuzz tests yet

## Capabilities

### New Capabilities

_None_ — this change is infrastructure only; it does not introduce or modify any
`swm` runtime capabilities.

### Modified Capabilities

_None_

## Impact

- New directory: `.github/workflows/`
- Requires GitHub repository secrets: `GHA_PAT_TOKEN`, `CACHIX_AUTH_TOKEN`
- Cachix cache name must be set to `swm` (or repo-appropriate name) in each workflow
- The `ci.yml` vendor-hash updater references the Nix package attribute path (e.g.
  `.#swm.goModules`) — must match the actual flake output name
- No changes to Go source, proto, or plugin interfaces
