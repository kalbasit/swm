## ADDED Requirements

### Requirement: OpenPane starts a program in a new pane

`session-tmux` SHALL implement `Session.OpenPane({workspace_id, pane_group_id,
argv, cwd, env})` by creating a new pane in the named pane group on the named
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

### Requirement: ListPanes enumerates panes across workspaces

`session-tmux` SHALL implement `Session.ListPanes({workspace_id,
pane_group_id})` by streaming one `Pane` per live pane.

- An empty `workspace_id` SHALL enumerate every live workspace socket under the
  socket directory, applying the same liveness probe `ListWorkspaces` uses, so
  that a single call answers what is running on the host.
- A non-empty `workspace_id` SHALL restrict the results to that workspace, and
  SHALL be reported as `NOT_FOUND` when no such socket exists.
- A non-empty `pane_group_id` SHALL restrict the results to panes in that pane
  group.
- Each streamed `Pane` SHALL carry `pane_id`, `pane_group_id`, and
  `workspace_id`, plus the pane title, the command currently running in it, its
  current working directory, and whether it is focused.
- A pane is `focused` when the multiplexer reports that it is the active pane of
  the active window of a session that has at least one attached client — that
  is, when a human's keystrokes would currently reach it.
- Results SHALL be ordered deterministically.

#### Scenario: All panes on the host

- **WHEN** two workspaces are live and `ListPanes({})` is called
- **THEN** panes from both workspaces are streamed, each carrying the workspace
  socket it came from

#### Scenario: Filtered by workspace

- **WHEN** `ListPanes({workspace_id: "<sock-a>"})` is called and both `<sock-a>`
  and `<sock-b>` are live
- **THEN** only panes from `<sock-a>` are streamed

#### Scenario: Filtered by pane group

- **WHEN** `ListPanes({workspace_id: "<sock>", pane_group_id: "github•com/kalbasit/swm"})` is called
- **THEN** only panes belonging to that pane group are streamed

#### Scenario: Focus reported

- **WHEN** a pane is the active pane of the active window of an attached session
- **THEN** its streamed `Pane` has `focused = true`, and panes that no attached
  client is currently typing into have `focused = false`

#### Scenario: Unknown workspace

- **WHEN** `ListPanes` names a workspace socket that does not exist
- **THEN** the call fails with `NOT_FOUND`

#### Scenario: Dead sockets skipped

- **WHEN** a socket file exists but its tmux server is no longer running and
  `ListPanes({})` is called
- **THEN** that socket contributes no panes and the call succeeds

### Requirement: SendText delivers text to a pane as if typed

`session-tmux` SHALL implement `Session.SendText({workspace_id, pane_id, text,
submit, delay_ms, allow_focused})` by delivering `text` to the pane and, when
`submit` is set, the Enter key after it.

- `text` SHALL be delivered literally. No part of it SHALL be interpreted as a
  key name, so text such as `Enter`, `C-c`, or a leading `-n` arrives as those
  characters and not as the corresponding key or a command flag.
- `submit` SHALL append the provider's submit key as a separate delivery.
- `delay_ms`, when positive, SHALL be observed *before* delivering, matching the
  semantics of `pane_cmd_delay` in the layout spec, and SHALL abort with the
  context's error if the context is cancelled while waiting.
- An empty `text` with `submit = true` SHALL be valid and deliver only the
  submit key, so a caller that needs a gap between text and submission can issue
  two calls.
- An empty `text` with `submit = false` SHALL be rejected with
  `INVALID_ARGUMENT`; there is nothing to deliver.
- An empty `workspace_id` or `pane_id` SHALL be rejected with
  `INVALID_ARGUMENT`.
- A `pane_id` that is not present on the workspace SHALL be reported as
  `NOT_FOUND`.

#### Scenario: Text and submit

- **WHEN** `SendText({workspace_id: "<sock>", pane_id: "%4", text: "hello", submit: true})` is called for an unfocused pane
- **THEN** the literal text `hello` is delivered to pane `%4`, followed by a
  separate Enter delivery

#### Scenario: Text that looks like a key name

- **WHEN** `SendText` is called with `text = "Enter"`
- **THEN** the five characters `Enter` are delivered, not the Enter key

#### Scenario: Text with a leading dash

- **WHEN** `SendText` is called with `text = "-n oops"`
- **THEN** the text is delivered as-is and is not interpreted as a flag

#### Scenario: Submit only

- **WHEN** `SendText` is called with an empty `text` and `submit = true`
- **THEN** only the Enter key is delivered

#### Scenario: Nothing to send

- **WHEN** `SendText` is called with an empty `text` and `submit = false`
- **THEN** the call fails with `INVALID_ARGUMENT`

#### Scenario: Delay observed before delivery

- **WHEN** `SendText` is called with `delay_ms = 200`
- **THEN** the delivery is issued no earlier than 200 ms after the call begins

#### Scenario: Unknown pane

- **WHEN** `SendText` names a `pane_id` that does not exist on the workspace
- **THEN** the call fails with `NOT_FOUND` and nothing is delivered

### Requirement: SendText refuses a focused pane unless overridden

Text delivered by `SendText` is indistinguishable from typing. A pane a human is
currently typing into may be displaying a confirmation prompt, and injected text
would answer it. `session-tmux` SHALL therefore refuse delivery when the target
pane is focused, as defined in the `ListPanes` requirement, and SHALL report
`FAILED_PRECONDITION` with a message naming `allow_focused` as the override.

When `allow_focused` is set, the plugin SHALL deliver regardless of focus. The
refusal is a guard against the common accident, not a security boundary: the
caller that sets `allow_focused` owns the consequences.

The focus determination SHALL come from the same provider query that populates
`Pane.focused`, so that a caller which sees `focused = false` from `ListPanes`
and immediately calls `SendText` is not refused for a reason it could not have
observed. The check is inherently racy — a human can focus the pane between the
two calls — and callers SHALL NOT treat a successful `SendText` as proof that no
human was typing.

#### Scenario: Focused pane refused

- **WHEN** `SendText` targets a pane that an attached client is currently
  focused on, without `allow_focused`
- **THEN** the call fails with `FAILED_PRECONDITION`, the message names
  `allow_focused`, and no text is delivered

#### Scenario: Focused pane with explicit override

- **WHEN** the same call is made with `allow_focused = true`
- **THEN** the text is delivered

#### Scenario: Unattached session is not focused

- **WHEN** `SendText` targets the active pane of a workspace that has no
  attached client
- **THEN** the pane is not treated as focused and the text is delivered

### Requirement: ClosePane terminates a pane idempotently

`session-tmux` SHALL implement `Session.ClosePane({workspace_id, pane_id})` by
killing the named pane on the named workspace socket.

A pane that no longer exists SHALL NOT be an error: the call SHALL succeed, so
that a caller cleaning up after a program that already exited does not have to
distinguish the two. An empty `workspace_id` or `pane_id` SHALL be rejected with
`INVALID_ARGUMENT`.

#### Scenario: Close a live pane

- **WHEN** `ClosePane({workspace_id: "<sock>", pane_id: "%4"})` is called and the
  pane exists
- **THEN** the pane is killed and the call succeeds

#### Scenario: Close a pane that is already gone

- **WHEN** `ClosePane` names a pane that no longer exists
- **THEN** the call succeeds

#### Scenario: Missing identifiers rejected

- **WHEN** `ClosePane` is called with an empty `workspace_id` or an empty
  `pane_id`
- **THEN** the call fails with `INVALID_ARGUMENT`
