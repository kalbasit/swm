# GitHub CI/CD Workflows

## Purpose

Defines the GitHub Actions workflows that automate continuous integration,
security scanning, dependency updates, releases, and backporting for the
repository.

## Requirements

### Requirement: CI runs on pull requests and pushes to primary branches
The CI workflow SHALL run `nix flake check` on every pull request targeting
`main` or `release-*`, and on every direct push to those branches.

#### Scenario: Pull request opened against main
- **WHEN** a pull request is opened, synchronized, or reopened targeting `main`
- **THEN** the CI workflow triggers and runs `nix flake check`

#### Scenario: Push to main branch
- **WHEN** a commit is pushed directly to `main`
- **THEN** the CI workflow triggers and runs `nix flake check`

#### Scenario: Concurrent runs are cancelled
- **WHEN** a new CI run starts for the same PR or branch
- **THEN** any in-progress run for that PR/branch is cancelled

---

### Requirement: Vendor hashes auto-update on go.mod changes
The CI workflow SHALL detect changes to `go.mod` or `go.sum` and, for non-fork
PRs, automatically update the `vendorHash` for every affected Go package and
commit the result back to the branch.

#### Scenario: go.mod changes on a non-fork PR
- **WHEN** `go.mod` or `go.sum` is modified in a non-fork pull request
- **THEN** the workflow runs `go mod tidy`, rebuilds each package's `goModules`
  derivation, extracts the new hash, updates `vendorHash` in the corresponding
  `nix/packages/<pkg>/default.nix`, and commits the changes back to the PR branch

#### Scenario: go.mod unchanged
- **WHEN** neither `go.mod` nor `go.sum` is modified
- **THEN** the vendor-hash update step is skipped

#### Scenario: Fork PR (no write access)
- **WHEN** the PR originates from a fork
- **THEN** the vendor-hash update step is skipped (cannot write to forks)

---

### Requirement: CI produces a single required status check
The CI workflow SHALL expose a single always-run job (`ci`) whose pass/fail
reflects the aggregate result of all other jobs, so branch protection rules
need only require one check.

#### Scenario: All jobs succeed
- **WHEN** all CI jobs complete successfully
- **THEN** the `ci` status check reports success

#### Scenario: Any required job fails
- **WHEN** any upstream CI job fails or errors
- **THEN** the `ci` status check reports failure

---

### Requirement: Semantic PR title is enforced
The semantic-PR workflow SHALL reject pull requests whose title does not conform
to the Conventional Commits specification.

#### Scenario: PR title follows Conventional Commits
- **WHEN** a PR is opened or its title is edited to match `<type>(<scope>): <desc>`
- **THEN** the check passes

#### Scenario: PR title does not follow Conventional Commits
- **WHEN** a PR title lacks a conventional commit prefix
- **THEN** the check fails and blocks merge

---

### Requirement: Security scan runs on schedule and on changes
The DevSkim workflow SHALL scan the repository for security issues on push to
`main`, on PRs targeting `main`, and on a weekly schedule. Results SHALL be
uploaded to the GitHub Security tab.

#### Scenario: Push to main triggers security scan
- **WHEN** code is pushed to `main`
- **THEN** DevSkim scans the repository and uploads SARIF results to GitHub Security

#### Scenario: Weekly scheduled scan
- **WHEN** the weekly cron schedule fires
- **THEN** DevSkim scans the repository and uploads SARIF results

---

### Requirement: Nix flake is updated weekly via automated PR
The flake-update workflow SHALL run weekly, update `flake.lock`, run
`go mod tidy`, open a PR, and enable auto-merge for that PR.

#### Scenario: Scheduled weekly flake update
- **WHEN** the weekly cron triggers
- **THEN** the workflow runs `nix flake update` and `go mod tidy`, commits to a
  `update-flake-lock` branch, opens a PR targeting `main`, and enables auto-merge

#### Scenario: Manual dispatch of flake update
- **WHEN** the workflow is triggered manually via `workflow_dispatch`
- **THEN** same behavior as the scheduled run

---

### Requirement: Releases are created on version tags
The release workflow SHALL trigger on `v*.*.*` tags, run `nix flake check`, and
create a draft GitHub Release with auto-generated notes and pre-release detection.

#### Scenario: Stable version tag pushed
- **WHEN** a tag matching `v*.*.*` (without alpha/beta/rc) is pushed
- **THEN** `nix flake check` runs and a draft GitHub Release is created as a stable release

#### Scenario: Pre-release version tag pushed
- **WHEN** a tag containing `alpha`, `beta`, or `rc` is pushed
- **THEN** `nix flake check` runs and a draft GitHub Release is created and marked as pre-release

---

### Requirement: Merged PRs can be backported to release branches
The backport workflow SHALL create backport PRs to `release-*` branches when a
merged PR is labeled with a `backport` label.

#### Scenario: Merged PR with backport label
- **WHEN** a PR is merged and carries a label beginning with `backport`
- **THEN** the workflow opens a backport PR against the indicated `release-*` branch
  with auto-merge (squash) enabled

#### Scenario: PR closed without merge
- **WHEN** a PR is closed but not merged
- **THEN** no backport PR is created

---

### Requirement: PRs must not have active OpenSpec changes
The CI workflow SHALL fail any pull request that has at least one non-archived
OpenSpec change present in `openspec/changes/` (i.e. any direct child
directory of `openspec/changes/` other than `archive`).

#### Scenario: PR has no active OpenSpec changes
- **WHEN** `openspec/changes/` contains no subdirectory other than `archive`
- **THEN** the `openspec-guard` job succeeds

#### Scenario: PR has one or more active OpenSpec changes
- **WHEN** `openspec/changes/` contains at least one subdirectory that is not
  `archive`
- **THEN** the `openspec-guard` job fails and blocks merge

#### Scenario: openspec/changes/ contains only the archive directory
- **WHEN** the only entry under `openspec/changes/` is the `archive` directory
- **THEN** the `openspec-guard` job succeeds

#### Scenario: openspec/changes/ is empty
- **WHEN** `openspec/changes/` exists but has no subdirectories
- **THEN** the `openspec-guard` job succeeds

#### Scenario: Guard failure is reflected in the aggregate ci check
- **WHEN** the `openspec-guard` job fails
- **THEN** the aggregate `ci` status check also fails, blocking merge without
  requiring an additional branch-protection rule
