## Why

Users have no way to discover which workspaces exist or what projects are attached to each without inspecting raw story JSON files. `swm workspace list` gives a quick, human-readable overview of the full workspace landscape.

## What Changes

- Add `swm workspace list` subcommand that prints a tree of all workspaces and their attached projects.
- Output is a pretty-printed tree: workspace names as top-level nodes, project paths indented beneath each.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `workflow-commands`: adds the `workspace list` subcommand to the existing `swm workspace` command group, alongside `workspace open`.

## Non-goals

- Machine-readable output (`--json`, `--porcelain`) — can be added later.
- Filtering or searching workspaces by name or project.
- Showing live session status (e.g. whether a tmux socket is active).
- Any changes to the plugin protocol or proto files.

## Impact

- `cmd/swm`: new `workspace list` cobra command wired under the existing `workspace` parent command.
- Reads from the story store (read-only, no mutations).
- No plugin invocations; no proto changes; no version bump required.
- Capability surface: **none** (host-only, no plugin involvement).
