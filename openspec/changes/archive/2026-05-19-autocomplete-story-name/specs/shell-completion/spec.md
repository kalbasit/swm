## ADDED Requirements

### Requirement: Story name completion for workspace open
The `swm workspace open` command SHALL provide dynamic shell completion for its optional `[story-name]` positional argument by returning all known story names from the story store.

#### Scenario: Story names are offered as completions
- **WHEN** the user presses Tab after `swm workspace open `
- **THEN** the shell displays all story names from the store as completion candidates

#### Scenario: Story name prefix is filtered by shell
- **WHEN** the user types `swm workspace open feat` and presses Tab
- **THEN** the shell displays only story names that begin with `feat`

#### Scenario: No fallback to filename completion
- **WHEN** the user presses Tab after `swm workspace open `
- **THEN** the shell does NOT offer filenames as completion candidates

#### Scenario: Store error degrades gracefully
- **WHEN** the user presses Tab after `swm workspace open ` and the story store returns an error
- **THEN** no completion candidates are offered and the shell exits without error output

### Requirement: Story name completion for story remove
The `swm story remove` command SHALL provide dynamic shell completion for its optional `[name]` positional argument by returning all known story names from the story store.

#### Scenario: Story names are offered as completions
- **WHEN** the user presses Tab after `swm story remove `
- **THEN** the shell displays all story names from the store as completion candidates

#### Scenario: Story name prefix is filtered by shell
- **WHEN** the user types `swm story remove feat` and presses Tab
- **THEN** the shell displays only story names that begin with `feat`

#### Scenario: No fallback to filename completion
- **WHEN** the user presses Tab after `swm story remove `
- **THEN** the shell does NOT offer filenames as completion candidates

#### Scenario: Store error degrades gracefully
- **WHEN** the user presses Tab after `swm story remove ` and the story store returns an error
- **THEN** no completion candidates are offered and the shell exits without error output
