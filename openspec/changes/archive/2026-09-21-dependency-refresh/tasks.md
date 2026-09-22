## 1. Nix flake inputs

- [x] 1.1 (no Go module) Record the pre-update baseline: current `pkgs.go.version` from the devshell and the locked rev/`lastModified` of every node in `flake.lock`, so the report can name exactly what moved.
- [x] 1.2 (no Go module) Run `nix flake update` and verify `flake.lock` changed for `nixpkgs`, `flake-parts`, `git-hooks-nix`, `treefmt-nix`, `process-compose-flake` and the transitive `flake-compat` node; note any input that did not move and why.
- [x] 1.3 (no Go module) Re-enter the devshell (`direnv` bootstrap) and verify whether the `go` directive in `go.work` and the 7 `go.mod` files was rewritten by the shellHook. Verify by `git diff -- go.work '*/go.mod'`: an empty diff means `pkgs.go` did not move. If it did move, record the old and new version for the report — do not revert it and do not raise it by hand.
- [x] 1.4 (no Go module) Run `task fmt` and verify it exits 0; commit any reformatting caused by a `treefmt-nix` bump as part of this change.
- [x] 1.5 (no Go module) Verify the flake still evaluates and builds: `nix build .#swm-full` exits 0. Vendor-hash failures at this point are expected only if §2 has already run; on a clean tree this must pass.

## 2. Go module dependencies

- [x] 2.1 (proto) Run `go get -u ./... && go mod tidy` in `proto/`; verify `go build ./...` and `go test ./...` exit 0 and that `grpc` reached v1.84.0 in `proto/go.mod`.
- [x] 2.2 (sdk/go) Same for `sdk/go/`; verify `go test ./...` exits 0. Confirm `hashicorp/go-plugin` and `go-hclog` stayed put (already current) and that no update is silently pinned back.
- [x] 2.3 (cmd/swm) Same for `cmd/swm/`; verify `gofrs/flock` v0.13.1, `golang.org/x/sys` v0.48.0, `golang.org/x/term` v0.46.0 and `grpc` v1.84.0 in `cmd/swm/go.mod`, and that `go test ./...` exits 0.
- [x] 2.4 (plugins/vcs-git, plugins/session-tmux, plugins/picker-fzf) Same per module; verify each module's `go test ./...` exits 0. In `plugins/picker-fzf/go.mod`, confirm the pseudo-version on `github.com/kalbasit/swm/sdk/go` is left alone — it is satisfied by the `replace` directive, not the proxy.
- [x] 2.5 (no Go module) Run `go list -m -u all` from any module and verify no *direct* dependency still shows an available update other than ones deliberately held; record each hold with its reason.

## 3. go-github major bump

- [x] 3.1 (plugins/forge-github) Rewrite the `github.com/google/go-github/v88` import path to `v92` in `internal/forge/github.go` and the require in `go.mod`, then `go mod tidy`. Verify `go build ./...` fails with exactly one error — `undefined: github.NewPullRequest` at the `PullRequests.Create` call site — confirming the break is the known one and nothing else.
- [x] 3.2 (plugins/forge-github) Port the call site: `&github.NewPullRequest{…}` → `github.CreatePullRequest{…}` (by value), with `Head` and `Base` as plain `string` from `req.GetHeadBranch()`/`req.GetBaseBranch()` instead of `new(…)` pointers. Verify `go build ./...` exits 0 and `go test ./...` passes with the existing `httptest`-based tests unmodified — the tests assert the outgoing request, so an unmodified pass is the evidence that wire behaviour is unchanged.
- [x] 3.3 (plugins/forge-github) Verify no `v88` reference survives anywhere: `grep -rn 'go-github/v88' .` returns nothing.

## 4. Nix vendor hashes

- [x] 4.1 (no Go module) Run `task update-nix-vendor-hashes` and verify it exits 0 and that the `vendorHash` line changed in each of the five `nix/packages/{swm,swm-plugin-forge-github,swm-plugin-picker-fzf,swm-plugin-session-tmux,swm-plugin-vcs-git}/default.nix` whose module graph moved. Do not hand-write a hash.
- [x] 4.2 (no Go module) Verify every Nix package builds against the new hashes: `nix build .#swm .#swm-full .#swm-plugin-forge-github .#swm-plugin-picker-fzf .#swm-plugin-session-tmux .#swm-plugin-vcs-git` exits 0.

## 5. Full verification

- [x] 5.1 (all modules) Run `task fmt` and verify exit 0 with a clean `git diff` afterwards.
- [x] 5.2 (all modules) Run `task lint` and verify exit 0 — a `golangci-lint` bump from nixpkgs can surface new findings that must be fixed, not suppressed.
- [x] 5.3 (all modules) Run `task test` and verify exit 0 across all 7 modules.
- [x] 5.4 (all modules) Run `nix flake check` and verify exit 0 — this is the only gate that exercises the vendor hashes, the single-module build sandbox, and the integration tests driving real plugin binaries over gRPC.

## 6. Report

- [x] 6.1 (no Go module) Write the dependency report: every flake input's old → new rev, every Go dependency's old → new version, every dependency deliberately held with its reason (at minimum the deprecated transitive `github.com/golang/protobuf`), and an explicit statement of whether the Go language version moved and why. Verify the report accounts for every line of `git diff --stat`.
