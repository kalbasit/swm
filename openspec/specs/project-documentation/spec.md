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
- **THEN** it SHALL contain: project name and one-line description, use-case summary, architecture overview (host + plugin model), quick-start install instructions, link to contributing guide, and links to per-module READMEs

### Requirement: Host CLI has a module-level README
`cmd/swm/README.md` SHALL document the host CLI in sufficient detail for a new user to install, configure, and run it.

#### Scenario: Host README covers CLI usage
- **WHEN** a user reads `cmd/swm/README.md`
- **THEN** it SHALL cover: available commands, configuration file format (TOML), plugin discovery order, and hook system overview

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
