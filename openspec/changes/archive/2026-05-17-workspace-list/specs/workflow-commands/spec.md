## ADDED Requirements

### Requirement: swm workspace list
- `swm workspace list` prints a tree of all workspaces and their attached projects to stdout.
- Workspaces are listed in lexicographic order by name.
- Projects within each workspace are listed in lexicographic order by their canonical path (`host/segments...`).
- The output uses a fixed tree glyph style:
  ```
  - <workspace-name>
    |
    |- <project-path>
    |- <project-path>
  - <workspace-name>
    |
    |- <project-path>
  ```
- Exit code is 0 on success, non-zero on store error.

#### Scenario: No workspaces
- **WHEN** `swm workspace list` is run and the story store contains no stories
- **THEN** the command exits zero and prints nothing to stdout

#### Scenario: Workspace with no projects
- **WHEN** `swm workspace list` is run and story `feat-x` exists with no attached projects
- **THEN** the command exits zero and prints `feat-x` as a top-level entry with no project children

#### Scenario: Single workspace with one project
- **WHEN** `swm workspace list` is run and story `feat-x` has one attached project `github.com/a/b`
- **THEN** the command exits zero and prints a tree with `feat-x` as the root and `github.com/a/b` indented beneath it

#### Scenario: Multiple workspaces with multiple projects
- **WHEN** `swm workspace list` is run and stories `alpha` and `beta` exist, with `alpha` having projects `github.com/a/b` and `github.com/c/d`, and `beta` having project `github.com/e/f`
- **THEN** the output lists `alpha` before `beta` (lexicographic), `github.com/a/b` before `github.com/c/d` within `alpha`, with the tree glyph style

#### Scenario: Store error
- **WHEN** `swm workspace list` is run and the story store returns an error
- **THEN** the command exits non-zero and prints the error to stderr
