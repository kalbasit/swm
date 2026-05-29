## MODIFIED Requirements

### Requirement: Clone to canonical path
The VCS plugin MUST implement `Clone` as a server-streaming RPC that emits progress during the clone and delivers the resolved `ProjectID` as the terminal stream event.

- The plugin runs `git clone --progress <url> <destination_path>`.
- Each `\r`- or `\n`-terminated segment from git's stderr is sent as a `CloneProgressEvent{progress_line: <segment>}`.
- On success the plugin sends a final `CloneProgressEvent{project_id: <id>}` and closes the stream.
- On failure the plugin returns a gRPC error status; no terminal `project_id` event is sent.
- If `destination_path` already contains a `.git` directory the plugin returns `codes.AlreadyExists` without running `git clone`.

#### Scenario: Successful clone emits progress then project ID
- **WHEN** `Clone({url: "git@github.com:kalbasit/swm.git", destination_path: "/tmp/code/repositories/github.com/kalbasit/swm"})` is called and the remote is reachable
- **THEN** one or more `CloneProgressEvent{progress_line: ...}` messages are streamed as git runs, followed by a final `CloneProgressEvent{project_id: {host: "github.com", segments: ["kalbasit", "swm"]}}`, after which the stream closes with no error

#### Scenario: Already cloned
- **WHEN** `Clone` is called with a `destination_path` that already contains a `.git` directory
- **THEN** a gRPC `AlreadyExists` status error is returned without running `git clone` and the stream carries no events

#### Scenario: git clone failure
- **WHEN** `git clone` exits non-zero (e.g., repository not found, auth error)
- **THEN** a gRPC `Internal` status error is returned with stderr captured in the message and the stream carries no terminal `project_id` event

## REMOVED Requirements

_(none — all other Clone to canonical path scenarios are superseded by the MODIFIED block above)_
