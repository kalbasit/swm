## ADDED Requirements

### Requirement: Renovate configuration file exists at repo root

A `renovate.json` file MUST exist at the repository root. It MUST be valid JSON conforming to the Renovate configuration schema and reference the `config:recommended` preset as the base.

#### Scenario: Config file present and parseable
- **WHEN** Renovate Bot scans the repository root
- **THEN** it finds `renovate.json`, parses it without errors, and activates dependency scanning

---

### Requirement: Go module updates covered for all modules

The Renovate configuration MUST enable the `gomod` package manager and MUST match all 7 `go.mod` files in the monorepo: `cmd/swm/go.mod`, `sdk/go/go.mod`, `proto/go.mod`, `plugins/vcs-git/go.mod`, `plugins/picker-fzf/go.mod`, `plugins/session-tmux/go.mod`, `plugins/forge-github/go.mod`.

Each Go module MUST receive its own PR per dependency update (no cross-module grouping).

#### Scenario: Upstream Go dependency releases new version
- **WHEN** a new version of a Go dependency is published that is used by one or more `go.mod` files
- **THEN** Renovate opens one PR per affected `go.mod` file, each updating only that module's dependency

#### Scenario: All go.mod files are discovered
- **WHEN** Renovate runs its discovery phase
- **THEN** all 7 `go.mod` files are identified as targets for update

---

### Requirement: Nix flake inputs covered

The Renovate configuration MUST enable the `nix` package manager to track flake input updates for the 5 inputs declared in `flake.nix`: `flake-parts`, `git-hooks-nix`, `nixpkgs`, `process-compose-flake`, `treefmt-nix`.

Each flake input MUST receive its own PR.

#### Scenario: Nix flake input publishes new commit
- **WHEN** a tracked flake input (e.g., `nixpkgs`) has a newer revision available
- **THEN** Renovate opens one PR updating only that input's `url` in `flake.nix`

---

### Requirement: GitHub Actions versions covered

The Renovate configuration MUST enable the `github-actions` package manager to track version pins in all workflow files under `.github/workflows/`.

#### Scenario: GitHub Action releases new version
- **WHEN** a GitHub Action used in a workflow file releases a new version tag
- **THEN** Renovate opens a PR updating the action's version pin in the affected workflow file

---

### Requirement: Weekly scheduling on Mondays

All Renovate PRs MUST be scheduled to open on Mondays only. No PRs may be opened outside the defined schedule window.

#### Scenario: Dependency update detected mid-week
- **WHEN** Renovate detects an available update on a Wednesday
- **THEN** it queues the update and opens the PR on the following Monday

---

### Requirement: Dependency PRs are labeled

All Renovate-opened PRs MUST have the labels `dependencies` and `renovate` applied automatically.

#### Scenario: Renovate opens a Go module update PR
- **WHEN** Renovate creates a PR for a Go dependency update
- **THEN** the PR has both the `dependencies` and `renovate` labels set

---

### Requirement: Automerge is disabled

No Renovate PR MUST be automerged. All updates MUST go through normal PR review and merge by a human.

#### Scenario: Renovate opens a minor version bump PR
- **WHEN** a dependency has a minor version update available
- **THEN** Renovate opens a PR but does NOT merge it automatically, regardless of CI status
