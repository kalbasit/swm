## MODIFIED Requirements

### Requirement: Releases are created on version tags
The release workflow SHALL trigger on `v*.*.*` tags, write the pushed tag into
`nix/packages/swm/version.txt` before building so the release build reports that
tag, run `nix flake check`, and create a draft GitHub Release with
auto-generated notes and pre-release detection. The tag SHALL be taken from the
pushed ref rather than from a value committed to the repository.

The trigger glob is looser than a semantic version, so the workflow SHALL
validate the pushed ref before building and SHALL fail rather than release a
build that cannot report its tag. A ref matched by the trigger but rejected by
that validation MUST NOT reach `nix flake check` or produce a release.
Pre-release and build-metadata identifiers MUST each be non-empty.

#### Scenario: Stable version tag pushed
- **WHEN** a tag matching `v*.*.*` (without alpha/beta/rc) is pushed
- **THEN** the tag is written to `nix/packages/swm/version.txt`, `nix flake check` runs, and a draft GitHub Release is created as a stable release

#### Scenario: Pre-release version tag pushed
- **WHEN** a tag containing `alpha`, `beta`, or `rc` is pushed
- **THEN** the tag is written to `nix/packages/swm/version.txt`, `nix flake check` runs, and a draft GitHub Release is created and marked as pre-release

#### Scenario: Tag matched by the trigger but not a semantic version
- **WHEN** a tag such as `v1.2.3.4`, `v1.2.3-rc..1`, or `v1.2.3+build.` is pushed
- **THEN** the workflow fails before `nix flake check` with an error naming the tag, and no release is produced
