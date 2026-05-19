## ADDED Requirements

### Requirement: PRs must not have active OpenSpec changes
The CI workflow SHALL fail any pull request that has at least one non-archived
OpenSpec change present in `openspec/changes/` (i.e. any direct child
directory of `openspec/changes/` other than `archive`).

#### Scenario: PR has no active OpenSpec changes
- **WHEN** `openspec/changes/` contains no subdirectory other than `archive`
- **THEN** the `openspec-guard` job succeeds

#### Scenario: PR has one or more active OpenSpec changes
- **WHEN** `openspec/changes/` contains at least one subdirectory that is not
  `archive`
- **THEN** the `openspec-guard` job fails and blocks merge

#### Scenario: openspec/changes/ contains only the archive directory
- **WHEN** the only entry under `openspec/changes/` is the `archive` directory
- **THEN** the `openspec-guard` job succeeds

#### Scenario: openspec/changes/ is empty
- **WHEN** `openspec/changes/` exists but has no subdirectories
- **THEN** the `openspec-guard` job succeeds

#### Scenario: Guard failure is reflected in the aggregate ci check
- **WHEN** the `openspec-guard` job fails
- **THEN** the aggregate `ci` status check also fails, blocking merge without
  requiring an additional branch-protection rule
