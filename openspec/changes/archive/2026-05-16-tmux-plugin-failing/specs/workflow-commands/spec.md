## ADDED Requirements

### Requirement: Plugin stderr forwarding
The host plugin manager SHALL forward each plugin process's stderr to the host's own
stderr so that plugin panics, `os.Exit` messages, and runtime errors are visible to the
operator without requiring a separate debug session.

Forwarding SHALL be enabled for every plugin capability (session, vcs, picker, forge)
and SHALL be set up at plugin launch time, before the first gRPC call is made.

#### Scenario: Plugin writes to stderr before crashing
- **WHEN** a plugin binary writes a message to its stderr and then exits non-zero
- **THEN** the message appears on the host's stderr, prefixed with the plugin binary path

#### Scenario: Plugin stderr forwarded for all capabilities
- **WHEN** swm launches a session, vcs, picker, or forge plugin
- **THEN** any output the plugin writes to stderr is forwarded to the host's stderr stream

#### Scenario: Healthy plugin produces no extra output
- **WHEN** a plugin runs successfully and writes nothing to stderr
- **THEN** the host's stderr receives no additional output from the plugin
