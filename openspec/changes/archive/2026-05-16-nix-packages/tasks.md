## 1. Scaffold Package Directory Structure

- [x] 1.1 Create `nix/packages/` directory with top-level `flake-module.nix` that imports all per-binary sub-modules and sets `packages.default = config.packages.swm-full`
- [x] 1.2 Create `nix/apps/` directory with `flake-module.nix` that exposes `apps.swm` and `apps.default` pointing to the `swm` binary inside `packages.swm-full`
- [x] 1.3 Add `./nix/packages/flake-module.nix` and `./nix/apps/flake-module.nix` to the `imports` list in `flake.nix`

## 2. Package: swm (cmd/swm)

- [x] 2.1 Create `nix/packages/swm/version.txt` (empty placeholder)
- [x] 2.2 Create `nix/packages/swm/default.nix` using `pkgs.buildGoModule`; fileset covers `cmd/swm/` (excluding `tests/integration/`), `proto/`, `sdk/go/`; set `pname = "swm"`, `doCheck = true`, `nativeBuildInputs = [ pkgs.git ]`, version from `version.txt` fallback to git rev
- [x] 2.3 Determine `vendorHash` for the `swm` module by running `nix build .#swm` with `vendorHash = lib.fakeHash` and updating the hash from the error output

## 3. Package: swm-plugin-forge-github (plugins/forge-github)

- [x] 3.1 Create `nix/packages/swm-plugin-forge-github/version.txt` (empty placeholder)
- [x] 3.2 Create `nix/packages/swm-plugin-forge-github/default.nix` using `pkgs.buildGoModule`; fileset covers `plugins/forge-github/`, `proto/`, `sdk/go/`; set `pname = "swm-plugin-forge-github"`, `doCheck = true` (tests use `net/http/httptest`, no external tools needed)
- [x] 3.3 Determine `vendorHash` for the `forge-github` module

## 4. Package: swm-plugin-picker-fzf (plugins/picker-fzf)

- [x] 4.1 Create `nix/packages/swm-plugin-picker-fzf/version.txt` (empty placeholder)
- [x] 4.2 Create `nix/packages/swm-plugin-picker-fzf/default.nix` using `pkgs.buildGoModule`; fileset covers `plugins/picker-fzf/`, `proto/`, `sdk/go/`; set `pname = "swm-plugin-picker-fzf"`, `doCheck = true` (tests build `fakefzf` from testdata using the Go compiler already in the build env; no extra `nativeBuildInputs`)
- [x] 4.3 Determine `vendorHash` for the `picker-fzf` module

## 5. Package: swm-plugin-session-tmux (plugins/session-tmux)

- [x] 5.1 Create `nix/packages/swm-plugin-session-tmux/version.txt` (empty placeholder)
- [x] 5.2 Create `nix/packages/swm-plugin-session-tmux/default.nix` using `pkgs.buildGoModule`; fileset covers `plugins/session-tmux/`, `proto/`, `sdk/go/`; set `pname = "swm-plugin-session-tmux"`, `doCheck = true` (tests build `faketmux` from testdata using the Go compiler; no real tmux needed; no extra `nativeBuildInputs`)
- [x] 5.3 Determine `vendorHash` for the `session-tmux` module

## 6. Package: swm-plugin-vcs-git (plugins/vcs-git)

- [x] 6.1 Create `nix/packages/swm-plugin-vcs-git/version.txt` (empty placeholder)
- [x] 6.2 Create `nix/packages/swm-plugin-vcs-git/default.nix` using `pkgs.buildGoModule`; fileset covers `plugins/vcs-git/`, `proto/`, `sdk/go/`; set `pname = "swm-plugin-vcs-git"`, `doCheck = true`, `nativeBuildInputs = [ pkgs.git ]` (tests invoke `git init/commit/etc.` via `os/exec`)
- [x] 6.3 Determine `vendorHash` for the `vcs-git` module

## 7. Aggregate Package: swm-full

- [x] 7.1 Create `nix/packages/swm-full/default.nix` using `pkgs.symlinkJoin` over all five per-binary packages; set `name = "swm-full"` and `paths = [ swm forge-github picker-fzf session-tmux vcs-git ]`
- [x] 7.2 Wire `swm-full` into `nix/packages/flake-module.nix` imports list alongside the five per-binary sub-modules

## 8. Verification

- [x] 8.1 Run `nix build .#swm` and verify `result/bin/swm` exists, is executable, and that the check phase (unit tests) passed
- [x] 8.2 Run `nix build .#swm-plugin-forge-github`, `.#swm-plugin-picker-fzf`, `.#swm-plugin-session-tmux`, `.#swm-plugin-vcs-git` and verify each produces the correct binary with tests passing
- [x] 8.3 Run `nix build .#swm-full` and verify all five binaries are present under `result/bin/`
- [x] 8.4 Run `nix build` (no attribute) and verify it builds `swm-full`
- [x] 8.5 Run `nix run .#swm -- --help` and verify the swm help text is printed
- [x] 8.6 Run `nix flake show` and verify `packages.<system>.swm`, `packages.<system>.swm-full`, and `apps.<system>.swm` appear for each supported system
- [x] 8.7 Run `task fmt && task lint` and confirm both exit with status 0
