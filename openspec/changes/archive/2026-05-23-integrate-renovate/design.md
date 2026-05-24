## Context

The swm monorepo contains 7 Go modules (`cmd/swm`, `sdk/go`, `proto`, `plugins/vcs-git`, `plugins/picker-fzf`, `plugins/session-tmux`, `plugins/forge-github`) and a Nix flake with 5 external inputs (`flake-parts`, `git-hooks-nix`, `nixpkgs`, `process-compose-flake`, `treefmt-nix`). GitHub Actions workflows also pin action versions. Currently all of these are updated manually, meaning security patches and upstream releases accumulate unnoticed between releases.

Renovate Bot (Mend Renovate) is the industry-standard automated dependency update tool. It reads a `renovate.json` config at the repository root, discovers dependency files, and opens focused PRs for updates on a schedule.

## Goals / Non-Goals

**Goals:**
- Automate PR creation for Go module updates across all 7 `go.mod` files.
- Automate PR creation for Nix flake input updates.
- Automate PR creation for GitHub Actions version pins.
- Apply consistent PR labels and scheduling to reduce noise.

**Non-Goals:**
- Automerging any dependency PR — all updates require human review.
- Covering non-Go/non-Nix dependency types (Docker, npm, etc.) — none currently exist in the repo.
- Configuring Renovate dashboards or issue tracking features.

## Decisions

### D1: Config location — `renovate.json` at repo root

Renovate's default discovery looks for `renovate.json`, `.renovaterc`, or `.renovaterc.json` at the root. Using `renovate.json` requires no extra discovery config and is the most widely-recognized location.

_Alternative considered_: `.github/renovate.json5` — avoids root clutter, but adds a non-standard path that requires explicitly telling teams where to look. Rejected for simplicity.

### D2: One PR per module for Go modules

Each of the 7 Go modules is an independent build unit with its own `go.mod`. Grouping them into a single PR would produce a large diff spanning unrelated packages and make bisection harder if a dependency breaks something.

_Alternative considered_: Group all minor/patch into one weekly PR — reduces PR count but conflates unrelated upgrades. Rejected.

### D3: One PR per Nix flake input

Each flake input (e.g. `nixpkgs`, `flake-parts`) evolves independently. A single PR per input keeps the change surface minimal and makes rollback straightforward.

### D4: Weekly schedule (Mondays)

Weekly batching avoids daily PR noise while still keeping dependencies reasonably current. Monday gives the team the work week to review and merge before any weekend incidents.

_Alternative considered_: Monthly — too infrequent for security patches. Rejected.

### D5: Automerge disabled

Given the repo's use of Nix vendored hashes and the `update-nix-vendor-hashes` task requirement, automated merges would require running `task update-nix-vendor-hashes` as part of the merge pipeline — complexity not worth the benefit at current team size.

## Risks / Trade-offs

- [Nix flake support] Renovate's `nix` manager is newer and may not parse all flake input formats → Mitigation: limit to known-working input styles; fall back to manual update for any that Renovate skips.
- [go.work monorepo] Renovate must be told to scan all `go.mod` files explicitly; default workspace heuristics may miss some → Mitigation: use explicit `fileMatch` patterns in the `gomod` manager config.
- [vendorHashes] Go module PRs will not automatically update Nix `vendorHash` values → Mitigation: document in the PR template that reviewers must run `task update-nix-vendor-hashes` after merging any Go dependency update.

## Migration Plan

1. Add `renovate.json` to the repository root on this branch.
2. Merge the PR — Renovate activates automatically on next scheduled run.
3. Renovate will open an onboarding PR if the GitHub App is not yet installed; install the app and approve.
4. No rollback needed — removing or emptying `renovate.json` disables Renovate silently.

## Open Questions

- Is the Mend Renovate GitHub App already installed on the `kalbasit` organization, or does it need to be added? (Check GitHub App settings before merging.)
