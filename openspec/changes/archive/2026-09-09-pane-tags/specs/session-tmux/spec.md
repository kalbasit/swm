## MODIFIED Requirements

### Requirement: OpenPane starts a program in a new pane

`session-tmux` SHALL implement `Session.OpenPane({workspace_id, pane_group_id,
argv, cwd, env, tags})` by creating a new pane in the named pane group on the named
workspace socket and returning a `Pane` whose `pane_id` is the identifier tmux
assigned to it.

- `argv` SHALL be run as the pane's program. It is an already-split argument
  vector; the plugin SHALL quote each element so the multiplexer's shell
  re-parses it into the same vector, and an element containing spaces or shell
  metacharacters SHALL NOT be split or expanded.
- An empty `argv` SHALL start the provider's default shell.
- `cwd`, when non-empty, SHALL be the pane's starting directory.
- `env` entries SHALL be set in the pane's environment. They SHALL be applied in
  a deterministic order so that repeated calls issue identical commands.
- The pane group SHALL be addressed by exact name, never by prefix or glob.
- `tags` SHALL be stored as pane-scoped tmux options, which is where tmux keeps
  what belongs to a pane rather than to the process in it. They SHALL therefore
  survive the pane's program exiting and being replaced, and SHALL cease to
  exist when the pane closes. They SHALL be applied in a deterministic order so
  repeated calls issue identical commands.

Where the new pane is placed is provider policy and is not part of the contract.

An empty `workspace_id` or `pane_group_id` SHALL be rejected with
`INVALID_ARGUMENT`. A pane group that does not exist SHALL be reported as
`NOT_FOUND`.

#### Scenario: Pane started with an argument vector

- **WHEN** `OpenPane({workspace_id: "<sock>", pane_group_id: "github•com/kalbasit/swm", argv: ["my-tool", "--flag", "two words"], cwd: "/tmp/wt"})` is called
- **THEN** a new pane is created in the `github•com/kalbasit/swm` pane group with
  starting directory `/tmp/wt`, running `my-tool --flag 'two words'`, and the
  returned `Pane` carries the identifier tmux assigned, the requested
  `pane_group_id`, and the requested `workspace_id`

#### Scenario: Empty argv starts a shell

- **WHEN** `OpenPane` is called with an empty `argv`
- **THEN** the pane is created with no explicit program and runs the default
  shell

#### Scenario: Environment is applied deterministically

- **WHEN** `OpenPane` is called with `env = {B: "2", A: "1"}`
- **THEN** the environment entries are applied in a stable order that does not
  depend on map iteration order

#### Scenario: Missing pane group

- **WHEN** `OpenPane` names a pane group that does not exist on the workspace
- **THEN** the call fails with `NOT_FOUND`

#### Scenario: Missing identifiers rejected

- **WHEN** `OpenPane` is called with an empty `workspace_id` or an empty
  `pane_group_id`
- **THEN** the call fails with `INVALID_ARGUMENT` and no tmux command is issued

#### Scenario: Tags outlive the program in the pane

- **WHEN** a pane is opened with tags and its program exits and is replaced
- **THEN** `ListPanes` still reports that pane's tags, because they were stored on the pane and not in the process

#### Scenario: Tags are read back by ListPanes

- **WHEN** `ListPanes` enumerates a pane that was opened with tags
- **THEN** the returned `Pane` carries them

