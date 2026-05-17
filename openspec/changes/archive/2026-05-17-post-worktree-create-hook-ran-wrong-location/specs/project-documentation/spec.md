## MODIFIED Requirements

### Requirement: Root README introduces the project
The repository root SHALL contain a `README.md` that gives a first-time visitor a clear understanding of the project within 60 seconds of reading.

#### Scenario: Root README covers minimum required sections
- **WHEN** a visitor reads `README.md` at the repo root
- **THEN** it SHALL contain: project name and one-line description, use-case summary, architecture overview (host + plugin model), quick-start install instructions, a brief hook system mention with a link to the hook system reference in `cmd/swm/README.md`, link to contributing guide, and links to per-module READMEs

### Requirement: Host CLI has a module-level README
`cmd/swm/README.md` SHALL document the host CLI in sufficient detail for a new user to install, configure, and run it.

#### Scenario: Host README covers CLI usage
- **WHEN** a user reads `cmd/swm/README.md`
- **THEN** it SHALL cover: available commands, configuration file format (TOML), plugin discovery order, and a Hook System section containing: all supported event names, the working directory each event runs in, tier resolution order (global → per-repo → per-story), the environment variables passed to each hook (`SWM_HOOK`, `SWM_STORY`, `SWM_PROJECT_HOST`, `SWM_PROJECT_PATH`, `SWM_WORKTREE_PATH`, `SWM_REPO_PATH`), and the stdin JSON contract

#### Scenario: Hook System section is reachable via anchor
- **WHEN** a reader follows a link to `cmd/swm/README.md#hook-system`
- **THEN** they land directly on the Hook System section
