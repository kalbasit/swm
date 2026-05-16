### Requirement: swm-full aggregate package merges all binaries
The flake SHALL expose `packages.swm-full`, a derivation produced by `pkgs.symlinkJoin`
that combines all five per-binary packages into a single store path.  The merged output
MUST contain every binary that the individual packages provide.

#### Scenario: All binaries reachable from swm-full
- **WHEN** a user runs `nix build .#swm-full`
- **THEN** `result/bin/swm`, `result/bin/swm-plugin-forge-github`, `result/bin/swm-plugin-picker-fzf`, `result/bin/swm-plugin-session-tmux`, and `result/bin/swm-plugin-vcs-git` are all present and executable

#### Scenario: swm-full is the default package
- **WHEN** a user runs `nix build` without specifying a package attribute
- **THEN** the build produces the same result as `nix build .#swm-full`

### Requirement: packages.default aliases swm-full
The flake SHALL set `packages.default = config.packages.swm-full` so that `nix build`
with no attribute selector builds the aggregate.

#### Scenario: nix build with no attribute
- **WHEN** a user runs `nix build` in the repository
- **THEN** all five swm binaries are present under `result/bin/`
