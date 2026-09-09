## ADDED Requirements

### Requirement: Story name completion for workspace ensure

The `swm workspace ensure` command SHALL provide dynamic shell completion for its optional `[story-name]` positional argument by returning all known story names from the story store.

#### Scenario: Story names are offered as completions

- **WHEN** the user presses Tab after `swm workspace ensure `
- **THEN** the shell displays all story names from the store as completion candidates

#### Scenario: No fallback to filename completion

- **WHEN** the user presses Tab after `swm workspace ensure `
- **THEN** the shell does NOT offer filenames as completion candidates

#### Scenario: Store error degrades gracefully

- **WHEN** the user presses Tab after `swm workspace ensure ` and the story store returns an error
- **THEN** no completion candidates are offered and the shell exits without error output
