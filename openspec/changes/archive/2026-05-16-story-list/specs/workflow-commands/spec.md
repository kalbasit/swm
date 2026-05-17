## ADDED Requirements

### Requirement: swm story list
`swm story list` SHALL print all story names to stdout, one per line, in
lexical order. The command takes no arguments and no flags. On success it exits
zero. If the store cannot be read it exits non-zero and prints an error to
stderr.

#### Scenario: Single story (default only)
- **WHEN** `swm story list` is run and only the `_default` story exists
- **THEN** the command exits zero and prints exactly `_default` to stdout

#### Scenario: Multiple stories
- **WHEN** `swm story list` is run and stories `alpha`, `beta`, and `_default` exist
- **THEN** the command exits zero and prints the names in lexical order, one per line

#### Scenario: Store error
- **WHEN** `swm story list` is run and `Store.List` returns an error
- **THEN** the command exits non-zero and prints a human-readable error message
