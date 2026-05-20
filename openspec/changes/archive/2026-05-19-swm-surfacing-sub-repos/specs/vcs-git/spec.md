## ADDED Requirements

### Requirement: ListProjects skips sub-repositories
When walking `repositories/` to discover projects, the host SHALL treat any directory
containing a `project_markers` entry (e.g. `.git`) as a leaf project and SHALL NOT
descend into that directory's subdirectories to look for additional projects.
Nested git repositories (e.g. tool-managed module caches, temporary clones) that live
inside a project directory MUST NOT be surfaced as independent projects.

#### Scenario: Nested tool-cache repos are suppressed
- **WHEN** `repositories/github.com/acme/infra/.git` exists AND one or more `.git`
  directories exist under `repositories/github.com/acme/infra/.terraform/modules/`
- **THEN** `ListProjects` streams exactly `{host: "github.com", segments: ["acme", "infra"]}`
  and does NOT stream any project whose path is under `repositories/github.com/acme/infra/`

#### Scenario: Sibling top-level repos are all returned
- **WHEN** `repositories/github.com/acme/app/.git` and
  `repositories/github.com/acme/infra/.git` both exist as direct project roots
- **THEN** `ListProjects` streams both `{host: "github.com", segments: ["acme", "app"]}`
  and `{host: "github.com", segments: ["acme", "infra"]}`

#### Scenario: Deep nested cache directories are suppressed
- **WHEN** `repositories/github.com/acme/infra/.git` exists AND multiple nested `.git`
  directories exist at arbitrary depths under `infra/` (e.g. `.terragrunt-cache/`,
  `tmp/`, vendor directories)
- **THEN** `ListProjects` streams exactly one project entry for `infra`
