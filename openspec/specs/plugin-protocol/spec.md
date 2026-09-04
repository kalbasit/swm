## Purpose

The protobuf contract between swm and its plugins: the module that owns all generated Go code, the services that contract defines, and how that code is generated and kept in step with the `.proto` sources.

## Requirements

### Requirement: Proto module owns all generated Go code

The `proto/` directory SHALL be its own Go module with path `github.com/kalbasit/swm/proto`. All `.proto` source files SHALL live under `proto/swm/plugin/v1/`. Generated Go code SHALL be written into the same module by `buf gen`. No other module SHALL contain hand-written copies of generated types.

#### Scenario: Proto module is importable

- **WHEN** a Go plugin or host module adds `require github.com/kalbasit/swm/proto` to its `go.mod`
- **THEN** the module compiles and the generated gRPC service client/server types are accessible

#### Scenario: Generated code is committed

- **WHEN** `task proto:gen` is run on a clean checkout
- **THEN** `git diff` shows no changes (generated output is deterministic and already committed)

### Requirement: Six proto services are defined

The `proto/swm/plugin/v1/` directory SHALL contain `.proto` files defining all six capability services: `Common`, `Session`, `VCS`, `Forge`, `Picker`, and `Host`. Each file SHALL compile without errors via `buf build`.

#### Scenario: buf build passes

- **WHEN** `buf build` is run inside `proto/`
- **THEN** the command exits 0 with no diagnostic output

#### Scenario: All six service names are present

- **WHEN** the generated Go package `pluginv1` is imported
- **THEN** the following interface types are defined: `SessionClient`, `SessionServer`, `VCSClient`, `VCSServer`, `ForgeClient`, `ForgeServer`, `PickerClient`, `PickerServer`, `HostClient`, `HostServer`

### Requirement: go_package option is set consistently

Every `.proto` file in `proto/swm/plugin/v1/` SHALL declare `option go_package = "github.com/kalbasit/swm/proto/swm/plugin/v1;pluginv1"`. No `.proto` file SHALL omit this option.

#### Scenario: buf lint passes

- **WHEN** `buf lint` is run inside `proto/`
- **THEN** the command exits 0 (all proto lint rules pass, including package naming and go_package consistency)

### Requirement: Proto generation is a Taskfile task with caching

A task `proto:gen` SHALL exist in `proto/tasks.yml`. It SHALL use `sources:` pointing at `*.proto` files and `generates:` pointing at the generated `.go` files so task skips regeneration when sources are unchanged.

#### Scenario: proto:gen is idempotent

- **WHEN** `task proto:gen` is run twice in sequence without modifying any `.proto` file
- **THEN** the second invocation exits 0 and prints "Task 'proto:gen' is up to date"

#### Scenario: proto:gen regenerates on proto change

- **WHEN** a `.proto` file is modified and `task proto:gen` is run
- **THEN** the generated `.go` files are updated to reflect the change

### Requirement: Session service exposes pane primitives

The `Session` service in `proto/swm/plugin/v1/session.proto` SHALL define four
pane-level RPCs alongside its existing workspace and pane-group RPCs:

```proto
rpc OpenPane(OpenPaneRequest) returns (Pane);
rpc ListPanes(ListPanesRequest) returns (stream Pane);
rpc SendText(SendTextRequest) returns (Empty);
rpc ClosePane(ClosePaneRequest) returns (Empty);
```

These are neutral multiplexer primitives. The vocabulary SHALL remain
provider-agnostic: the wire contract SHALL NOT name any provider-specific
mechanism (for example `send-keys`), and SHALL NOT model what a pane is being
used for (for example an agent, a job, or a build). A `Pane` is an addressable
place where a program runs; what runs there is the caller's concern.

The additions SHALL be additive: no existing RPC, message, or field number in
`session.proto` is changed or removed.

#### Scenario: Pane RPCs are present on the Session service

- **WHEN** the generated Go package `pluginv1` is imported
- **THEN** `SessionClient` and `SessionServer` both declare `OpenPane`,
  `ListPanes`, `SendText`, and `ClosePane`, and `ListPanes` is server-streaming

#### Scenario: Existing Session RPCs are unchanged

- **WHEN** `session.proto` is compared against the previous revision
- **THEN** `Info`, `OpenWorkspace`, `CloseWorkspace`, `ListWorkspaces`,
  `OpenPaneGroup`, `SwitchTo`, `IsInsideWorkspace`, and `CurrentContext` retain
  their request and response types, and every pre-existing message field keeps
  its field number

#### Scenario: buf lint and buf build pass

- **WHEN** `buf lint` and `buf build` are run inside `proto/`
- **THEN** both exit 0

### Requirement: Pane identifiers are opaque provider handles

`Pane.pane_id` SHALL be a handle minted by the session plugin whose format is
provider-specific (for example `%4` for `session-tmux`). Callers SHALL pass it
back verbatim to `SendText` and `ClosePane` and SHALL NOT parse, construct,
compare structurally, or otherwise derive meaning from it. The proto SHALL
document this constraint on the field.

This preserves an invariant the `Session` service already relies on:
`SwitchToRequest.close_origin_pane_id` is documented as "the
multiplexer-specific pane reference (e.g. `$TMUX_PANE` for session-tmux)", so an
opaque pane handle already crosses this boundary. The pane primitives formalise
that existing practice rather than introducing a new kind of value.

A `pane_id` SHALL be meaningful only within the workspace that produced it,
which is why every request carrying a `pane_id` also carries a `workspace_id`.

#### Scenario: Handle round-trips without interpretation

- **WHEN** a caller receives a `Pane` from `OpenPane` or `ListPanes` and later
  passes its `pane_id` unmodified to `SendText` or `ClosePane` together with the
  `workspace_id` it came from
- **THEN** the plugin addresses that same pane

#### Scenario: Opacity is documented at the field

- **WHEN** `proto/swm/plugin/v1/session.proto` is read
- **THEN** the comment on `Pane.pane_id` states that the value is
  provider-specific and that callers must not parse it

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

#### Scenario: Identity fields are always populated

- **WHEN** a plugin returns a `Pane` from `OpenPane` or `ListPanes`
- **THEN** `pane_id`, `pane_group_id`, and `workspace_id` are all non-empty

#### Scenario: A provider without a title reports the zero value

- **WHEN** a provider cannot report a pane title
- **THEN** `Pane.title` is the empty string and the call still succeeds

### Requirement: SendText declares the focused-pane hazard in the contract

`SendTextRequest` SHALL carry `text`, a `submit` flag indicating whether the
provider's submit key follows the text, a `delay_ms` grace period observed
before delivery, and an `allow_focused` opt-out.

The proto SHALL document that text delivered to a pane arrives as if typed, so
it can be consumed by whatever prompt the pane is currently displaying —
including a confirmation a human is in the middle of answering — and that
plugins refuse delivery into a focused pane unless `allow_focused` is set.

`delay_ms` SHALL be documented as the same mitigation the `session-tmux` layout
spec already exposes as `pane_cmd_delay`: a wait observed *before* delivery, so
a program that has just started has time to install its input handler.

#### Scenario: Hazard and opt-out are documented

- **WHEN** `proto/swm/plugin/v1/session.proto` is read
- **THEN** the comments on `SendTextRequest` state that text arrives as if typed,
  that a focused pane is refused by default, and that `allow_focused` overrides
  the refusal
