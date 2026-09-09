## MODIFIED Requirements

### Requirement: Version falls back to git revision
Each package SHALL determine its version as follows:
1. Read `version.txt` from the package's `nix/packages/<name>/` directory.
2. If the file is non-empty (after trimming whitespace), use that value.
3. Otherwise use `self.rev or self.dirtyRev` from the flake self reference.

Each package SHALL inject that version into the binary it builds at link time,
so the derivation's version and the version the binary reports can never
disagree. A package MUST NOT rely on a default compiled into the Go source.

#### Scenario: Clean release build
- **WHEN** `version.txt` contains `v1.2.3`
- **THEN** the built binary reports version `v1.2.3`

#### Scenario: Development build without a version tag
- **WHEN** `version.txt` is empty
- **THEN** the derivation uses the git commit SHA as the version, and the built binary reports that same SHA

#### Scenario: Plugin reports its build version
- **WHEN** a plugin package is built and the host asks the plugin for its info
- **THEN** the plugin reports the version its derivation was built with, not a placeholder

## ADDED Requirements

### Requirement: The swm package verifies the version reached the binary
`packages.swm` SHALL fail the build if the installed binary does not report the
derivation's version, so the wiring cannot silently regress. The check MUST run
only when the built binary can be executed on the build machine.

#### Scenario: Version reaches the binary
- **WHEN** `packages.swm` is built natively
- **THEN** the build runs `swm --version` and succeeds only if the output names the derivation's version

#### Scenario: Cross-compiled build
- **WHEN** `packages.swm` is cross-compiled for a foreign platform
- **THEN** the check is skipped rather than failing on an unrunnable binary
