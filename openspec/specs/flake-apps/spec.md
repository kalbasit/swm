### Requirement: Flake exposes apps.swm and apps.default
The flake SHALL expose two application entries so that `nix run` works without a package
selector:
- `apps.swm` — runs the `swm` binary from `packages.swm-full`
- `apps.default` — alias for `apps.swm`

Each app entry MUST set `program` to the path of the `swm` binary inside the
`swm-full` derivation using `lib.getExe` or an equivalent explicit path.

#### Scenario: nix run launches the swm binary
- **WHEN** a user runs `nix run` in the repository
- **THEN** the `swm` CLI starts and prints its help output

#### Scenario: nix run with explicit app name
- **WHEN** a user runs `nix run .#swm`
- **THEN** the `swm` CLI starts and prints its help output

### Requirement: Flake imports packages and apps modules
The `flake.nix` `imports` list SHALL include `./nix/packages/flake-module.nix` and
`./nix/apps/flake-module.nix` so both the `packages` and `apps` outputs are registered
with flake-parts.

#### Scenario: flake evaluation includes package outputs
- **WHEN** a user runs `nix flake show`
- **THEN** the output includes `packages.<system>.swm`, `packages.<system>.swm-full`, and `apps.<system>.swm` for each supported system

#### Scenario: Package and apps modules are imported in flake.nix
- **WHEN** `flake.nix` is evaluated
- **THEN** both `./nix/packages/flake-module.nix` and `./nix/apps/flake-module.nix` are in the `imports` list
