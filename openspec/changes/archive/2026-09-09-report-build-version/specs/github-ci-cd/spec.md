## MODIFIED Requirements

### Requirement: Releases are created on version tags
The release workflow SHALL trigger on `v*.*.*` tags, write the pushed tag into
`nix/packages/swm/version.txt` before building so the release build reports that
tag, run `nix flake check`, and create a draft GitHub Release with
auto-generated notes and pre-release detection. The tag SHALL be taken from the
pushed ref rather than from a value committed to the repository.

#### Scenario: Stable version tag pushed
- **WHEN** a tag matching `v*.*.*` (without alpha/beta/rc) is pushed
- **THEN** the tag is written to `nix/packages/swm/version.txt`, `nix flake check` runs, and a draft GitHub Release is created as a stable release

#### Scenario: Pre-release version tag pushed
- **WHEN** a tag containing `alpha`, `beta`, or `rc` is pushed
- **THEN** the tag is written to `nix/packages/swm/version.txt`, `nix flake check` runs, and a draft GitHub Release is created and marked as pre-release

#### Scenario: Ref that is not a version tag
- **WHEN** the workflow runs for a ref that does not match a version tag pattern
- **THEN** `version.txt` is left empty and the build falls back to the flake revision
