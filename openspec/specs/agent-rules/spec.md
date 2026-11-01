# Spec: Agent Rules

## Purpose

Defines the Claude Code rules swm ships under `contrib/claude-rules/`, which keep an agent session
confined to one swm story and make it resolve that story from its working directory.

## Requirements

### Requirement: Contrib ships the story rules as top-level Markdown files

The repository SHALL ship `contrib/claude-rules/swm-story-confinement.md` and
`contrib/claude-rules/swm-story-from-path.md`. Every rule SHALL be a `.md` file directly under
`contrib/claude-rules/`, not in a subdirectory, so that an installer that links every top-level
`.md` file there into `~/.claude/rules` picks up all of them.

#### Scenario: Both rules are present at the top level

- **WHEN** the files directly under `contrib/claude-rules/` are listed
- **THEN** `swm-story-confinement.md` and `swm-story-from-path.md` are both present
- **AND** no rule file sits in a subdirectory

### Requirement: Rules are worded for swm only

Each rule SHALL refer to swm only, through `$SWM_STORY`, `swm config get`, `swm story attach`,
`swm workspace open` and the story JSON under `${XDG_DATA_HOME:-$HOME/.local/share}/swm/stories/`.
A rule SHALL NOT name another story manager's commands or environment variables, and SHALL NOT
contain host names, user names or organisation-specific paths, other than links to swm's own
public repository.

#### Scenario: No foreign tool names

- **WHEN** the rule files are searched case-insensitively for another story manager's name
- **THEN** there are no matches

### Requirement: Confinement uses the path-resolved story

`swm-story-confinement.md` SHALL take the session's story from the story that
`swm-story-from-path.md` resolved from the working directory when that rule is installed, and SHALL
fall back to `$SWM_STORY` only when it is not installed. Within the confinement rule, `$SWM_STORY`
SHALL mean that resolved story.

#### Scenario: Inherited story differs from the working directory

- **WHEN** both rules are installed, the working directory is under `$CODE_ROOT/stories/A/`, and the
  inherited `$SWM_STORY` is `B`
- **THEN** the confinement rule scopes the session to story `A`

#### Scenario: Confinement installed alone

- **WHEN** only `swm-story-confinement.md` is installed
- **THEN** the rule reads the story from `$SWM_STORY`

### Requirement: README lists every contrib rule

The root README's Contrib table SHALL have a row for every file in `contrib/claude-rules/`, linking
to it and saying what it does.

#### Scenario: Table matches the directory

- **WHEN** the rule files are compared with the README Contrib table
- **THEN** each rule file has exactly one row
