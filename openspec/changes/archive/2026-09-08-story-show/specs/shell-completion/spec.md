## ADDED Requirements

### Requirement: Story name completion for story show

The `swm story show` command SHALL provide dynamic shell completion for its optional `[story-name]` positional argument by returning all known story names from the story store.

#### Scenario: Story names are offered as completions

- **WHEN** the user presses Tab after `swm story show `
- **THEN** the shell displays all story names from the store as completion candidates

#### Scenario: No fallback to filename completion

- **WHEN** the user presses Tab after `swm story show `
- **THEN** the shell does NOT offer filenames as completion candidates

#### Scenario: Store error degrades gracefully

- **WHEN** the user presses Tab after `swm story show ` and the story store returns an error
- **THEN** no completion candidates are offered and the shell exits without error output
