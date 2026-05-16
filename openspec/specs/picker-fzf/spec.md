### Requirement: Picker.Pick bidirectional streaming RPC
The `picker-fzf` plugin SHALL implement the `Picker.Pick` bidirectional gRPC streaming RPC. The host streams `PickItem` messages (each with a `display` string and an opaque `key`), and the plugin streams back a single `PickResult` message containing the `key` of the item the user selected.

#### Scenario: Single item selected
- **WHEN** the host streams five `PickItem` messages and the user selects one item in fzf
- **THEN** the plugin streams exactly one `PickResult` with the `key` of the selected item and closes the response stream

#### Scenario: User cancels (no selection)
- **WHEN** the host streams candidates and the user presses `Escape` or `Ctrl-C` in fzf
- **THEN** the plugin returns a gRPC `Aborted` status and streams no `PickResult`

### Requirement: fzf subprocess with TTY attachment
The `picker-fzf` plugin SHALL launch fzf as a subprocess with `/dev/tty` opened as both stdin and stdout, so that the fzf TUI renders in the user's terminal. Candidates received from the gRPC stream SHALL be accumulated in memory and written to fzf's stdin pipe before fzf is launched.

#### Scenario: fzf renders in terminal
- **WHEN** `Pick` is called with a non-empty candidate list in a terminal with a valid TTY
- **THEN** the fzf interactive selector is displayed in the terminal and the user can navigate and select items

#### Scenario: Non-TTY fallback
- **WHEN** `Pick` is called in an environment where `/dev/tty` cannot be opened (e.g., CI, pipe)
- **THEN** the plugin returns a gRPC `FailedPrecondition` status with a message indicating no TTY is available

### Requirement: Info RPC declares picker capability
The `picker-fzf` plugin SHALL implement the `Info` RPC, returning a `PickerInfo` with `plugin_info.name = "fzf"`, the build version, and no required or optional capability dependencies.

#### Scenario: Info response
- **WHEN** the host calls `Info()` on the picker-fzf plugin
- **THEN** a `PickerInfo` is returned with `plugin_info.name = "fzf"`
