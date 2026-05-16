# Tasks: github-workflows

## 1. Repository Setup

- [x] 1.1 Create a Cachix cache named `swm` and obtain the `CACHIX_AUTH_TOKEN`
- [x] 1.2 Add GitHub repository secrets: `GHA_PAT_TOKEN`, `CACHIX_AUTH_TOKEN`
- [x] 1.3 Create `.github/workflows/` directory in the repository

## 2. CI Workflow

- [x] 2.1 Create `.github/workflows/ci.yml` with `filter` job detecting `go_deps` changes
- [x] 2.2 Add `generate` job to `ci.yml` — loops over all five Go packages, runs `go mod tidy`, updates each `vendorHash` in `nix/packages/<pkg>/default.nix`, commits back via `stefanzweifel/git-auto-commit-action`
- [x] 2.3 Add `flake-check` job to `ci.yml` — runs `nix flake check -L` (handles fmt, lint, tests)
- [x] 2.4 Add `ci` always-run aggregate job to `ci.yml` — fails if any upstream job failed
- [x] 2.5 Configure concurrency group to cancel in-progress runs on the same PR/branch

## 3. Semantic PR Workflow

- [x] 3.1 Create `.github/workflows/semantic-pull-request.yml` targeting `master` and `release-*`

## 4. DevSkim Security Scan

- [x] 4.1 Create `.github/workflows/devskim.yml` targeting `master` branch; include weekly cron schedule

## 5. Flake Update Workflow

- [x] 5.1 Create `.github/workflows/flake-update.yml` — runs `nix flake update` + `go mod tidy`, commits to `update-flake-lock` branch, opens PR targeting `master`, enables auto-merge (squash)

## 6. Release Workflow

- [x] 6.1 Create `.github/workflows/releases.yml` — triggers on `v*.*.*` tags, runs `nix flake check`, creates a draft GitHub Release with pre-release detection for alpha/beta/rc

## 7. Backport Workflow

- [x] 7.1 Create `.github/workflows/backport.yml` targeting `release-*` branches with `korthout/backport-action`, squash auto-merge enabled

## 8. Validation

- [x] 8.1 Open a test PR and verify CI workflow triggers and the `ci` status check appears
- [x] 8.2 Verify semantic-PR check fires on a PR with a non-conforming title
- [x] 8.3 Verify the flake-update workflow can be manually dispatched without error
