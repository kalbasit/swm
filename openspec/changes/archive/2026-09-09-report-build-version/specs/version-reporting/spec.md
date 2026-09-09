## Purpose

Defines the version string the `swm` binary reports, so that an operator
looking at a running host can tell which build it is without probing for
subcommands, and so a build that is not a release can never be mistaken for
one.

## ADDED Requirements

### Requirement: swm reports the version it was built from
`swm --version` SHALL report the version the binary was built from. The version
SHALL be determined at build time, never read from a checked-in file at run
time, and the binary MUST NOT carry a hardcoded release-shaped default.

#### Scenario: Version flag output
- **WHEN** a user runs `swm --version`
- **THEN** swm writes `swm version <version>` to stdout and exits `0`

#### Scenario: Release build
- **WHEN** the binary was built with the link-time version set to `v1.2.3`
- **THEN** `swm --version` reports `v1.2.3`

### Requirement: Version resolution order
The version SHALL be resolved from the first of the following sources that
yields a non-empty value:

1. The version injected at link time by the build (the Nix packages inject the
   tag or the flake revision).
2. The main module's version recorded by the Go toolchain, when it is a real
   module version rather than the `(devel)` placeholder — this is what
   `go install <module>@<tag>` records.
3. The VCS revision recorded by the Go toolchain, rendered as `devel+<revision>`
   and suffixed with `-dirty` when the toolchain recorded uncommitted changes.
4. The literal `devel`, when no build information is available at all.

#### Scenario: Link-time version wins over build info
- **WHEN** the binary was built with the link-time version set to `v1.2.3` and the Go toolchain also recorded a module version
- **THEN** `swm --version` reports `v1.2.3`

#### Scenario: Installed with go install at a tag
- **WHEN** no link-time version was injected and the Go toolchain recorded main module version `v1.2.3`
- **THEN** `swm --version` reports `v1.2.3`

#### Scenario: Built from a clean working tree without a tag
- **WHEN** no link-time version was injected, the main module version is the `(devel)` placeholder, and the Go toolchain recorded revision `abc123` with no modifications
- **THEN** `swm --version` reports `devel+abc123`

#### Scenario: Built from a modified working tree
- **WHEN** no link-time version was injected and the Go toolchain recorded revision `abc123` with uncommitted modifications
- **THEN** `swm --version` reports `devel+abc123-dirty`

#### Scenario: No build information available
- **WHEN** no link-time version was injected and no Go build information is available
- **THEN** `swm --version` reports `devel`

### Requirement: Non-release builds are visibly not releases
A version reported by a build that was not given a release tag SHALL NOT look
like a release version. Such a version MUST be either a bare commit revision or
a `devel`-prefixed string, never a `vX.Y.Z` string.

#### Scenario: Untagged Nix build
- **WHEN** a package is built from a checkout with no release tag injected
- **THEN** the reported version is the commit revision, suffixed with `-dirty` when the working tree was dirty, and is not of the form `vX.Y.Z`
