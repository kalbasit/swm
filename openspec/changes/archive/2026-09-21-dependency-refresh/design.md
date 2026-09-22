## Context

See `proposal.md` — Why.

Three facts about this repo shape the approach:

1. **Go version is derived, not declared.** `nix/devshells/flake-module.nix` has a `shellHook` that rewrites the `go <version>` directive in `go.work` and every `go.mod` to `${pkgs.go.version}` on every direnv load, under a `flock` on `go.work.lock`. The `go` directive is therefore an *output* of the pinned nixpkgs, not something a human edits. Any attempt to hand-pin it is reverted on the next shell entry.
2. **Nix builds pin the module graph twice.** Each `nix/packages/*/default.nix` carries a literal `vendorHash`. `buildGoModule` runs `go mod vendor`, so any `go.sum` movement invalidates all five hashes and `nix build` / `nix flake check` fail until `task update-nix-vendor-hashes` refreshes them.
3. **It is a Go workspace.** `go.work` unifies all 7 modules, so `go mod tidy` must run per-module (from each module dir) to write each `go.mod`/`go.sum`, but `go build`/`go test` resolve against the workspace.

## Goals / Non-Goals

**Goals:**

- One commit that leaves `nix build`, `nix flake check`, `task fmt`, `task lint`, and `task test` all green.
- Each dependency movement attributable: what moved, or why it did not.

**Non-Goals:**

- Splitting this into per-dependency commits. Renovate's one-PR-per-dependency policy exists for unattended bumps; this is a deliberate batch refresh, and the go-github bump plus the vendor-hash refresh are not independently committable anyway.
- Pinning any dependency backwards to avoid work. A dependency held at its old version must be held for a stated upstream reason, not for convenience.

## Decisions

**Order the work Nix-first, then Go.** `nix flake update` can move `pkgs.go`, which would rewrite every `go` directive and change what `go mod tidy` resolves. Running Nix first means the Go work happens against the final toolchain. For this refresh both the old and new `nixpkgs` ship Go 1.26.7, so the order is a precaution rather than a necessity — but it is the correct order to establish.

*Alternative considered:* Go first, then Nix. Rejected — a toolchain bump landing second could silently invalidate a just-tidied `go.sum`.

**Take the go-github major bump rather than holding at v88.** The repository uses a deliberately thin slice of go-github: `NewClient`/`ClientOptionsFunc`/`WithAuthToken`, `PullRequests.Create`, `PullRequests.List` with `PullRequestListOptions`, `PullRequests.Get`, and field reads on `*github.PullRequest`. A scratch-tree bisect showed v89 compiles untouched and v90 introduces the only break: `github.NewPullRequest` → `github.CreatePullRequest`, passed by value, with `Head` and `Base` as plain `string`. The fix is four lines at one call site and the existing `httptest`-based tests pass unmodified — this is an accompanying source change to keep the build green, not a redesign.

*Alternative considered:* hold at v88 and report the break. Rejected — the change is mechanical and bounded, and holding would leave the repo four majors behind with nothing automated to close the gap.

**Let `task update-nix-vendor-hashes` compute the hashes; do not hand-edit them.** The script mirrors CI's `generate` job (build `.#<pkg>.goModules`, scrape `got:`, patch in place). Hand-writing a hash is how a stale one gets committed.

**Verify through `nix flake check`, not only `task test`.** `task test` uses the Go workspace; the Nix packages build each module in isolation with vendored deps and `doCheck = true`. Only `nix flake check` exercises the vendor hashes and the single-module sandboxing, which is exactly what a dependency change can break.

## Risks / Trade-offs

- **`treefmt-nix` or `git-hooks-nix` bump changes formatter output** → run `task fmt` immediately after the flake update and commit any reformatting as part of this change; a pre-commit hook failure then surfaces before the Go work, not during the commit.
- **`nixpkgs` bump moves `pkgs.go`** → the devshell rewrites all `go` directives automatically. Detect it by diffing `go.work`/`go.mod` after re-entering the shell, and report it explicitly rather than presenting it as a deliberate language-version raise. (Confirmed not to happen for this refresh: 1.26.7 both sides.)
- **go-github v92 changes wire behaviour beyond the rename** → the `forge-github` tests assert the outgoing request against an `httptest` server, so a changed request body or path fails the suite rather than escaping to GitHub.
- **A `grpc` v1.84.0 regression reaches the plugin transport** → covered by the integration tests under `cmd/swm/tests/integration`, which drive real plugin binaries over gRPC via `nix flake check`.
- **Batching means a single bisect point if something breaks later** → accepted; the commit body will enumerate every version delta so a follow-up revert can be surgical.
