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
