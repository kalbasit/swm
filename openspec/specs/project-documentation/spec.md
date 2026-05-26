# Spec: Project Documentation

## Purpose

This capability covers the documentation artifacts that must exist in the repository so that first-time visitors, users, and plugin authors can understand, install, configure, and extend the project.

## Requirements

### Requirement: Repository root has an Apache 2.0 LICENSE file
The repository root SHALL contain a `LICENSE` file with the full text of the Apache License, Version 2.0, with copyright assigned to Wael Nasreddine and the current year.

#### Scenario: LICENSE file is present and correct
- **WHEN** a visitor views the repository root
- **THEN** a `LICENSE` file SHALL be present containing "Apache License" and "Version 2.0"

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

### Requirement: Go SDK has a module-level README
`sdk/go/README.md` SHALL document the Go SDK for plugin authors.

#### Scenario: SDK README covers plugin authoring
- **WHEN** a plugin author reads `sdk/go/README.md`
- **THEN** it SHALL cover: how to implement each capability interface, how to register the plugin, and how to declare capability dependencies

### Requirement: Each bundled plugin has a README
Each directory under `plugins/` SHALL contain a `README.md` covering that plugin's purpose, configuration, and usage.

#### Scenario: Plugin README covers minimum sections
- **WHEN** a user reads any `plugins/<name>/README.md`
- **THEN** it SHALL contain: purpose, runtime requirements, configuration keys (with types and defaults), example config snippet, and known limitations

#### Scenario: All four bundled plugins have READMEs
- **WHEN** the repository is inspected
- **THEN** `plugins/forge-github/README.md`, `plugins/picker-fzf/README.md`, `plugins/session-tmux/README.md`, and `plugins/vcs-git/README.md` SHALL each exist

### Requirement: session-tmux README documents layout configuration

The `plugins/session-tmux/README.md` SHALL include a "Layout configuration" section that
documents the built-in TOML-based layout engine.

#### Scenario: Two-tier config lookup is explained
- **WHEN** a user reads `plugins/session-tmux/README.md`
- **THEN** it SHALL describe the priority order: per-repo `.swm/session-tmux.toml` wins over
  global `$XDG_CONFIG_HOME/swm/session-tmux.toml`, falling back to a default shell pane when
  neither exists

#### Scenario: Template-variable table is present and complete
- **WHEN** a user reads `plugins/session-tmux/README.md`
- **THEN** it SHALL contain a table listing `{{.WorktreePath}}`, `{{.StoryName}}`, and
  `{{.TmuxSocket}}` with a description of each

#### Scenario: TOML schema reference is present
- **WHEN** a user reads `plugins/session-tmux/README.md`
- **THEN** it SHALL contain a commented TOML block showing all top-level fields, `[[windows]]`,
  `[[windows.panes]]`, and nested `[[windows.panes.panes]]` entries with inline descriptions

#### Scenario: At least one complete example is present
- **WHEN** a user reads `plugins/session-tmux/README.md`
- **THEN** it SHALL contain at least one full `session-tmux.toml` example
