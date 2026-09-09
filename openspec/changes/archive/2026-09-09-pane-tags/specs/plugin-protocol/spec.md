## MODIFIED Requirements

### Requirement: Pane message carries identity plus best-effort description

The `Pane` message SHALL carry `pane_id`, `pane_group_id`, and `workspace_id`,
which together locate the pane. It SHALL additionally carry descriptive fields
reported by the provider — a title, the command currently running, the current
working directory, and whether the pane is focused — so that `ListPanes` can
answer "what is running on this host right now" without the caller inspecting
the multiplexer itself.

The descriptive fields SHALL be documented as best-effort: a provider that
cannot report one SHALL leave it at its zero value, and callers SHALL NOT treat
them as authoritative identity.

The `Pane` message SHALL additionally carry the tags the pane was opened with.
Tags are opaque to swm: they are a caller's own marks on a pane, so that a
caller can recognise a pane it opened without having kept the pane id, and after
the program in that pane has exited and been replaced. They belong to the pane
and SHALL NOT be derived from whatever is running in it.

#### Scenario: Identity fields are always populated

- **WHEN** a plugin returns a `Pane` from `OpenPane` or `ListPanes`
- **THEN** `pane_id`, `pane_group_id`, and `workspace_id` are all non-empty

#### Scenario: A provider without a title reports the zero value

- **WHEN** a provider cannot report a pane title
- **THEN** `Pane.title` is the empty string and the call still succeeds

#### Scenario: Tags are reported as they were given

- **WHEN** a pane is opened with tags and later returned by `ListPanes`
- **THEN** the `Pane` carries those tags unchanged

#### Scenario: A pane opened with no tags reports none

- **WHEN** a pane is opened with no tags
- **THEN** the `Pane` carries no tags, rather than an empty entry a caller must interpret

