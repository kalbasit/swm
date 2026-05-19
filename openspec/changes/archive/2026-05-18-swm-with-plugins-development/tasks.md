# Tasks: swm-with-plugins-development

## 1. SWM_PLUGIN_PATH support in cmd/swm

- [x] 1.1 (`cmd/swm`) Write failing table-driven tests in `pluginmgr/manager_test.go` covering: SWM_PLUGIN_PATH wins over PATH, SWM_PLUGIN_PATH wins over explicit config, colon-separated list searched left-to-right, non-existent entries silently skipped, unset SWM_PLUGIN_PATH leaves existing discovery unchanged
- [x] 1.2 (`cmd/swm`) In `pluginmgr/manager.go`, read `SWM_PLUGIN_PATH` at discovery time, split on `:`, skip non-existent/non-directory entries, and prepend matching dirs to the search list ahead of all existing tiers
- [x] 1.3 (`cmd/swm`) Run `task test` scoped to `cmd/swm` and confirm all new tests pass

## 2. scripts/swm-dev wrapper

- [x] 2.1 Create `scripts/swm-dev`: bash script that resolves `REPO_ROOT` via `git rev-parse --show-toplevel`, builds `cmd/swm` and all four plugins (`forge-github`, `picker-fzf`, `session-tmux`, `vcs-git`) into `$REPO_ROOT/.dev-bin/` using `go build`, sets `SWM_PLUGIN_PATH=$REPO_ROOT/.dev-bin`, then `exec`s `$REPO_ROOT/.dev-bin/swm "$@"`
- [x] 2.2 Make `scripts/swm-dev` executable (`chmod +x`) and add `.dev-bin/` to `.gitignore`
- [x] 2.3 Manually verify `./scripts/swm-dev --help` succeeds from the repo root and from a subdirectory

## 3. Nix devshell PATH update

- [x] 3.1 In `nix/devshells/flake-module.nix`, add `export PATH="$PWD/scripts:$PATH"` to the `shellHook` so `swm-dev` is on PATH inside direnv without a path prefix

## 4. Verification

- [x] 4.1 Run `task fmt` and apply any formatting changes
- [x] 4.2 Run `task lint` and fix any issues
- [x] 4.3 Run `task test` and confirm all tests pass
