## 1. Version resolution package (`cmd/swm`)

- [x] 1.1 Add `cmd/swm/internal/version` with `Resolve(linked string, info *debug.BuildInfo) string` returning the link-time value when it is non-empty; verify with a table-driven test that a linked `v1.2.3` wins even when `info` also carries a module version
- [x] 1.2 Extend `Resolve` to fall back to the main module version, ignoring the `(devel)` placeholder; verify with tests for a recorded `v1.2.3` and for `(devel)` falling through
- [x] 1.3 Extend `Resolve` to fall back to `devel+<vcs.revision>`, suffixed `-dirty` when `vcs.modified` is `true`; verify with tests for both the clean and dirty settings
- [x] 1.4 Extend `Resolve` to return `devel` when `info` is nil or carries nothing usable; verify with a test for a nil `*debug.BuildInfo`
- [x] 1.5 Add `Version()` applying `Resolve` to the unexported link-time variable and `debug.ReadBuildInfo()`; verify `go test ./cmd/swm/...` passes and that a test asserts `Version()` returns a non-empty string that is not a `vX.Y.Z` string in an untagged test build

## 2. Wire the CLI (`cmd/swm`)

- [x] 2.1 Replace the hardcoded `var version = "v2.0.0-dev"` in `cmd/swm/main.go` with `root.Version = version.Version()`; verify `grep -R "v2.0.0-dev" cmd/` returns nothing and `go build ./cmd/swm` succeeds
- [x] 2.2 Confirm the flag output shape by running the freshly built binary with `--version`; verify it no longer prints `v2.0.0-dev` but a `devel`-prefixed string. Note: this checkout is a git worktree, whose `.git` is a file rather than a directory, so the Go toolchain records no VCS settings and the build reports plain `devel`; a build from an ordinary clone stamps the revision and reports `devel+<sha>`

## 3. Nix packaging

- [x] 3.1 Add `ldflags = [ "-X github.com/kalbasit/swm/cmd/swm/internal/version.version=${version}" ]` to `nix/packages/swm/default.nix`; verify `nix build .#swm` succeeds and `./result/bin/swm --version` reports the flake revision
- [x] 3.2 Add the version assertion to the `swm` package's platform-guarded `postInstall`, comparing `$out/bin/swm --version` against `swm version ${version}` and failing the build on mismatch; verify by temporarily corrupting the expected string that the build fails, then restoring it and confirming `nix build .#swm` passes
- [x] 3.3 Add the matching `ldflags` line to each of the four plugin packages, targeting each plugin's `buildVersion` var by its full module path; verify `nix build .#swm-plugin-forge-github .#swm-plugin-picker-fzf .#swm-plugin-session-tmux .#swm-plugin-vcs-git` succeeds
- [x] 3.4 Confirm a tagged build path works end to end by writing `v9.9.9` into `nix/packages/swm/version.txt`, running `nix build .#swm`, checking `./result/bin/swm --version` reports `swm version v9.9.9`, then restoring the file to empty

## 4. Release workflow

- [x] 4.1 Add a "Create version file" step to the `flake-check` job in `.github/workflows/releases.yml` that writes `github.ref_name` into `nix/packages/swm/version.txt` when the ref matches the version-tag pattern, placed after checkout and before `nix flake check`; verify the step's regex accepts `v2.7.1` and `v2.7.1-rc.1` and rejects a branch name by running it locally against those values

## 5. Verification

- [x] 5.1 Run `task fmt`, `task lint`, and `task test` and confirm each exits `0`
- [x] 5.2 Run `openspec validate report-build-version --strict` and confirm it passes
