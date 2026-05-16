## ADDED Requirements

### Requirement: SDK is a standalone Go module

The `sdk/go/` directory SHALL be its own Go module with path `github.com/kalbasit/swm/sdk/go`. It SHALL import `github.com/kalbasit/swm/proto` for generated types. No SDK package SHALL import `cmd/swm` or any plugin module.

#### Scenario: SDK module compiles independently

- **WHEN** `go build ./...` is run inside `sdk/go/`
- **THEN** the command exits 0 without referencing any module outside `sdk/go`, `proto`, or the standard library + declared dependencies

### Requirement: One package per capability with interface and Serve helper

The SDK SHALL provide four packages: `session`, `vcs`, `forge`, and `picker` under `sdk/go/`. Each package SHALL export:
- A Go interface type that a plugin implementation must satisfy (e.g. `session.Plugin`, `vcs.Plugin`)
- A `Serve(impl Plugin) error` function that starts the go-plugin gRPC server for that capability

#### Scenario: Session interface is satisfied by an empty struct

- **WHEN** a Go type implements all methods declared by `session.Plugin`
- **THEN** it compiles as an argument to `session.Serve()`

#### Scenario: Serve returns error on missing implementation

- **WHEN** `session.Serve(nil)` is called
- **THEN** the function returns a non-nil error (does not panic)

### Requirement: SDK defines the go-plugin handshake constants

A package `sdk/go/handshake` SHALL export the `plugin.HandshakeConfig` value used by both the host and all plugins. Both sides SHALL import this package to ensure the cookie and protocol version match.

#### Scenario: Handshake constant is shared

- **WHEN** both host and plugin import `github.com/kalbasit/swm/sdk/go/handshake`
- **THEN** they use the same `MagicCookieKey`, `MagicCookieValue`, and `ProtocolVersion` values without copying them

### Requirement: SDK does not expose global state

No SDK package SHALL declare package-level variables that are mutated at runtime. All state required by `Serve()` SHALL be passed as arguments or held in unexported structs.

#### Scenario: Multiple Serve calls in one process do not interfere

- **WHEN** two capability `Serve()` functions are called sequentially in the same test binary
- **THEN** neither call affects the other's state
