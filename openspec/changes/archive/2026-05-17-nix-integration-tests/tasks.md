## 1. Go: env-var injection in TestMain

- [x] 1.1 In `cmd/swm/tests/integration/main_test.go`, for each of the six binaries (`vcsGitBin`, `sessionTmuxBin`, `faketmuxBin`, `pickerFzfBin`, `fakefzfBin`, `forgeGithubBin`), add an env-var check at the top of `TestMain` — if the corresponding `SWM_PLUGIN_*_BIN` / `SWM_TEST_*_BIN` var is non-empty, assign it directly and skip the `buildBin` call
- [x] 1.2 Run `go test ./tests/integration/...` locally (without env vars) to confirm the fallback `go build` path still works

## 2. Nix: faketmux and fakefzf packages

- [x] 2.1 Create `nix/packages/swm-test-faketmux/default.nix` — `buildGoModule` with `modRoot = "plugins/session-tmux"`, `subPackages = [ "internal/session/testdata/faketmux" ]`, same source fileset as `swm-plugin-session-tmux`, `doCheck = false`
- [x] 2.2 Create `nix/packages/swm-test-faketmux/version.txt` with contents `""` (empty, uses git rev)
- [x] 2.3 Create `nix/packages/swm-test-fakefzf/default.nix` — `buildGoModule` with `modRoot = "plugins/picker-fzf"`, `subPackages = [ "internal/picker/testdata/fakefzf" ]`, same source fileset as `swm-plugin-picker-fzf`, `doCheck = false`
- [x] 2.4 Create `nix/packages/swm-test-fakefzf/version.txt` with contents `""` (empty, uses git rev)
- [x] 2.5 Add `./swm-test-faketmux` and `./swm-test-fakefzf` to the `imports` list in `nix/packages/flake-module.nix`
- [x] 2.6 Run `nix build .#swm-test-faketmux` and `nix build .#swm-test-fakefzf` and fix the `vendorHash` values reported by Nix

## 3. Nix: integration check derivation

- [x] 3.1 In `nix/checks/flake-module.nix`, add `checks.swm-integration-tests` as a `buildGoModule` derivation:
  - `modRoot = "cmd/swm"`
  - Source fileset: full `cmd/swm` (no exclusion of `tests/integration`), plus `proto` and `sdk/go`
  - `vendorHash`: same value as `packages.swm` (add a cross-reference comment)
  - `subPackages = []` (no binary to install)
  - `buildPhase = ":"` (skip building)
  - `installPhase = "mkdir -p $out"` (empty output)
  - `doCheck = true`
  - `nativeBuildInputs = [ pkgs.git ]`
  - `preCheck`: set all six env vars from `self'.packages.*` outputs; also set `XDG_RUNTIME_DIR` and `HOME` to temp dirs
  - `checkFlags = [ "-v" "-run" "." "./tests/integration/..." ]`
- [x] 3.2 Run `nix build .#checks.x86_64-linux.swm-integration-tests` (or the appropriate system) and fix any issues

## 4. Spec update

- [x] 4.1 In `openspec/specs/nix-packages/spec.md`, apply the delta from `openspec/changes/nix-integration-tests/specs/nix-packages/spec.md`: replace the "swm unit tests pass" scenario to remove "tests/integration/ is absent" and replace with "exercised separately by checks.swm-integration-tests"

## 5. Verification

- [x] 5.1 Run `nix flake check` and confirm `checks.swm-integration-tests` passes
- [x] 5.2 Run `task fmt && task lint && task test` and confirm all exit 0
