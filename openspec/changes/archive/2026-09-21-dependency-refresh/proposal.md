## Why

Both halves of the toolchain have drifted. `flake.lock` trails current `nixpkgs`/`git-hooks-nix`/`flake-parts` (`treefmt-nix`, `process-compose-flake` and the transitive `flake-compat` are checked too, but are already at upstream HEAD), and the Go module graph is behind on `grpc`, three `golang.org/x/*` modules, `gofrs/flock`, and — most significantly — `google/go-github`, which is four major versions behind (v88 vs v92). Renovate opens one PR per dependency per module and never proposes major Go bumps, so the go-github gap will not close on its own.

Capability surface affected: **forge** (go-github sits behind the `forge-github` plugin). No proto changes.

## What Changes

- `nix flake update`: re-resolve all five declared flake inputs plus the transitive `flake-compat` node; only `nixpkgs`, `git-hooks-nix` and `flake-parts` are behind and actually move. Current `nixpkgs` and latest `nixpkgs` both ship Go **1.26.7**, so this does not move the Go toolchain.
- Go module dependencies across all 7 modules brought to current:
  - `google.golang.org/grpc` v1.83.2 → v1.84.0 (all 7 modules)
  - `github.com/gofrs/flock` v0.13.0 → v0.13.1 (`cmd/swm`)
  - `golang.org/x/sys` v0.47.0 → v0.48.0, `golang.org/x/term` v0.45.0 → v0.46.0 (`cmd/swm`)
  - indirect: `golang.org/x/net` v0.59.0, `golang.org/x/text` v0.42.0, `genproto/googleapis/rpc`
- **BREAKING (upstream, contained)** `github.com/google/go-github` v88 → v92 in `plugins/forge-github`. v90 removed `github.NewPullRequest`; the replacement is `github.CreatePullRequest`, passed **by value**, with `Head` and `Base` as plain `string` rather than `*string`. One call site (`internal/forge/github.go:112`). Verified in a scratch tree: a four-line edit, build and tests green. No swm-facing behaviour change — the same PR-create request is sent.
- Refresh `vendorHash` in all five `nix/packages/*/default.nix` via `task update-nix-vendor-hashes`, since `go.sum` moves.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. No requirement in `openspec/specs/` changes behaviour: the go-github bump is an upstream API rename behind an unchanged `forge-github` contract, and the `github-ci-cd` spec already requires weekly flake updates and automatic vendor-hash refresh. This change sets `skip_specs: true` rather than inventing a requirement.

## Non-goals

- Raising the Go language version. The devshell rewrites the `go` directive in `go.work` and every `go.mod` to `pkgs.go.version`; it stays 1.26.7 because nixpkgs' Go has not moved.
- Writing a spec for that devshell go-directive sync. It is real, load-bearing and currently unspecified, but documenting it is its own change.
- Moving `github.com/golang/protobuf` (deprecated). It is transitive via `hashicorp/go-plugin`; upstream owns it.
- Changing Renovate config, CI workflows, or the `forge` proto contract.

## Impact

- Files: `flake.lock`, all 7 `go.mod`/`go.sum`, `plugins/forge-github/internal/forge/github.go`, five `nix/packages/*/default.nix`.
- Risk concentrates in `treefmt-nix`/`git-hooks-nix` (could shift `nix fmt` or pre-commit output) and in the go-github bump. Both are covered by `task fmt` / `task lint` / `task test` plus `nix flake check`.
