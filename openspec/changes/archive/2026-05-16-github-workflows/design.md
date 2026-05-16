# Design: GitHub Workflows

## Context

`swm` has no CI/CD today. The `ncps` project has a mature workflow set using Nix +
Cachix + GitHub Actions. `swm` shares the same Nix-based toolchain, so adapting
those workflows is low-effort and high-value.

Notable differences from ncps:
- Default branch is `main` (not `main`)
- No database, no Docker images, no Helm chart, no docs site
- Five separate Go packages each with their own vendorHash in nix/packages/
- No existing Cachix cache — one must be created and named (convention: `swm`)

## Goals / Non-Goals

**Goals**
- Automated lint/test/build on every PR and push to `main`/`release-*`
- Automated vendor hash updates when `go.mod`/`go.sum` change
- GitHub Releases on `v*.*.*` tags
- Semantic PR title enforcement
- Security scanning via DevSkim
- Weekly Nix flake + `go mod tidy` automation
- Bot-driven backports to `release-*` branches

**Non-Goals**
- Docker image builds or Helm chart releases
- Fuzz testing infrastructure
- Docs deployment
- Codecov integration (no coverage config yet)

## Decisions

### 1. Default branch: `main`

The repository's canonical default branch is `main`. All workflows target
`main` (and `release-*`) instead of `main` as in ncps.

**Why**: `origin/HEAD → origin/main` per git config; a workflow targeting `main`
only would never fire on the primary branch.

### 2. Vendor hash update strategy: loop all packages

`swm` has six independent Go packages, each with its own `vendorHash` in its
`nix/packages/<pkg>/default.nix`. The ncps approach (single `sed` replacement in
one file) cannot be applied verbatim.

**Decision**: In the generate CI job, loop over all five package names, run
`nix build .#<pkg>.goModules` per package, extract the new hash from stderr, and
`sed` the matching file.

**Packages requiring vendor hash updates**:
- `swm`
- `swm-plugin-forge-github`
- `swm-plugin-picker-fzf`
- `swm-plugin-session-tmux`
- `swm-plugin-vcs-git`

(`swm-full` is a meta-package with no Go source of its own.)

**Alternative considered**: A single shared `vendorHash` — rejected because each
plugin has different transitive Go dependencies.

### 3. CI gate via `nix flake check`

Rather than separate lint/test/fmt steps, `nix flake check` runs all devshell
hooks (nixfmt, golangci-lint, tests). This is already proven to work locally.

**Alternative considered**: Running `task fmt`, `task lint`, `task test` directly —
rejected because it duplicates what `nix flake check` already does and requires
a heavier devshell activation in CI.

### 4. Cachix cache name: `swm`

No Cachix cache exists yet. The cache name `swm` matches the repo name convention.
The `CACHIX_AUTH_TOKEN` secret must be provisioned before the workflows can push
to the cache.

### 5. Release workflow: flake check + GitHub Release only

`ncps` releases build multi-arch Docker images then create a GitHub Release.
`swm` has no container artifacts, so the release workflow is:
`nix flake check` → `gh release create` (draft, with pre-release detection).

### 6. Semantic PR: adopt as-is

The `amannn/action-semantic-pull-request` action is project-agnostic. Targeting
`main` (and `release-*`) is the only adaptation needed.

### 7. Flake update: adapt for swm, no sqlc/go generate

The ncps flake-update workflow runs `nix flake update` + `go mod tidy` +
`sqlc generate` + `go generate`. For swm, omit the sqlc/go-generate steps.
The vendor hash update (if go.mod changes) is handled by the CI `generate` job
triggered by the flake-update PR, not inline in the flake-update workflow itself.

## Risks / Trade-offs

- **Cachix secret not yet provisioned** → CI cache steps will fail (non-blocking
  for correctness, but slow). Mitigation: set up Cachix org and secret before
  merging. Workflows should degrade gracefully if the cache miss happens.
- **Five-package vendor-hash loop is verbose** → Mitigation: encapsulate in a
  reusable shell script called from the workflow step.
- **`release-*` branch pattern assumed** → `swm` has no release branches yet.
  Backport and multi-branch workflows are forward-compatible and harmless until
  `release-*` branches are created.

## Open Questions

- What Cachix org / token name should be used? (Assumed `swm` — confirm before
  setting up the GitHub secret.)
- Should the release workflow push Nix packages to Cachix on tag? (Not implemented
  now; easy to add later by extending `releases.yml`.)
