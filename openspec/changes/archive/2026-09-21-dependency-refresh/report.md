# Dependency refresh report

Verified green: `task fmt` (0 changed), `task lint` (0 issues, 7 modules), `task test` (7 modules incl. integration), `nix flake check` (11 checks, x86_64-linux).

## Go language version: unchanged, deliberately

Stayed at **1.26.7**. The `go` directive is not hand-maintained: the devshell
`shellHook` in `nix/devshells/flake-module.nix` rewrites it in `go.work` and all
7 `go.mod` files to `${pkgs.go.version}` on every shell entry, under a flock on
`go.work.lock`. Current and updated `nixpkgs` both ship Go 1.26.7, so re-entering
the shell after `nix flake update` produced an empty diff on those 8 files. No
dependency forced a language-version raise.

## Nix flake inputs

| Input | Old | New |
|---|---|---|
| `nixpkgs` | `2bc7fd48` (2026-08-25) | `d0c92d84` (2026-09-22) |
| `git-hooks-nix` | `809414f0` (2026-08-22) | `59f4ca0d` (2026-09-15) |
| `flake-parts` | `9d0d8717` (2026-08-24) | `31729ca8` (2026-09-03) |

Not moved — already at upstream HEAD, confirmed against `nix flake metadata`:

| Input | Rev | Reason |
|---|---|---|
| `treefmt-nix` | `27b3b12a` | already current |
| `process-compose-flake` | `464ff688` | already current |
| `flake-compat` | `5edf11c4` | already current; transitive via `git-hooks-nix` |

## Go dependencies updated

| Module path | Old | New | Direct in |
|---|---|---|---|
| `google.golang.org/grpc` | v1.83.2 | v1.84.0 | all 7 modules |
| `github.com/google/go-github` | v88.0.0 | **v92.0.0** | `plugins/forge-github` |
| `github.com/gofrs/flock` | v0.13.0 | v0.13.1 | `cmd/swm` |
| `golang.org/x/term` | v0.45.0 | v0.46.0 | `cmd/swm` |
| `golang.org/x/sys` | v0.47.0 | v0.48.0 | `cmd/swm` (indirect elsewhere) |
| `golang.org/x/net` | v0.58.0 | v0.59.0 | indirect |
| `golang.org/x/text` | v0.41.0 | v0.42.0 | indirect |
| `google.golang.org/genproto/googleapis/rpc` | `20260825221802-da73d73af1c5` | `20260921155816-b14227669459` | indirect |

After the refresh, **zero of the 14 direct dependencies across all 7 modules has
an available update** (cross-checked `go list -m -u all` against the non-indirect
requires of every `go.mod`).

### The one breaking change: go-github v88 → v92

Four majors at once, because Renovate never proposes major Go bumps and this gap
would not close on its own. Bisected in a scratch tree rather than assumed:

- **v89** compiles untouched.
- **v90** introduced the only break: `github.NewPullRequest` was replaced by
  `github.CreatePullRequest`, passed **by value** rather than as a pointer, with
  `Head` and `Base` as plain `string` instead of `*string`.

One call site, `plugins/forge-github/internal/forge/github.go:112`. The port is
four lines. `internal/forge/github_test.go` is **unmodified** and passes — those
tests assert the outgoing request against an `httptest` server, so an unmodified
pass is the evidence that the wire behaviour of `CreatePullRequest` is unchanged.
No swm-facing behaviour change; the `forge` capability contract is untouched.

## Held back

| Dependency | Held at | Reason |
|---|---|---|
| `github.com/golang/protobuf` | v1.5.4 | Deprecated upstream, but **transitive** — pulled by `hashicorp/go-plugin` (via `ptypes/empty`) and by our own `cmd/swm/internal/pluginmgr` through it. Not ours to move; retiring it needs `go-plugin` to drop it first. |
| `github.com/kalbasit/swm/sdk/go` pseudo-version in `plugins/picker-fzf/go.mod` | `v0.0.0-20260825223552-d3cff0a71d07` | `go get -u` bumped this to a newer same-repo commit; reverted deliberately. It is satisfied by a `replace` directive pointing at `../../sdk/go`, so the version string is never used for resolution — churning it would imply a dependency moved when none did. |
| ~25 deep transitives (otel, `cloud.google.com/go/*`, `envoyproxy/*`, `spiffe`, `googleapis/gax-go`, …) | various | Not in any `go.mod` require block. They appear in `go list -m all`'s module graph via grpc's optional/test surface but are not part of our build; `go mod tidy` correctly does not record them. |

## Files changed (21) — all accounted for

- `flake.lock` — the 3 input bumps above
- 7 × `go.mod` + 7 × `go.sum` — the dependency table above. `go.sum` files shrink
  net (−272/+132 overall) because `go mod tidy` pruned entries left stale by
  earlier updates.
- 5 × `nix/packages/*/default.nix` — `vendorHash` recomputed by
  `task update-nix-vendor-hashes` (never hand-written). The
  `swm-test-faketmux`, `swm-test-fakefzf` and integration-test derivations derive
  their hash from `config.packages.<pkg>.goModules.outputHash`, so these 5 literal
  hashes are the complete set.
- `plugins/forge-github/internal/forge/github.go` — the go-github v92 call-site port
