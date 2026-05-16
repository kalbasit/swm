## Why

The repository has no `LICENSE` file and no `README.md`, which blocks public release: potential contributors cannot determine the terms of use, and visitors have no orientation to the project's purpose or structure. These omissions must be resolved before the project can be shared or adopted externally.

## What Changes

- Add `LICENSE` at the repository root — Apache License 2.0, copyright Wael Nasreddine.
- Add `README.md` at the repository root: high-level project introduction, use-cases, architecture overview, plugin system summary, and contributing guide.
- Add `cmd/swm/README.md`: host CLI details — commands, config format, plugin discovery, hook system.
- Add `sdk/go/README.md`: Go SDK usage, implementing a plugin, interface contracts.
- Add `plugins/forge-github/README.md`, `plugins/picker-fzf/README.md`, `plugins/session-tmux/README.md`, `plugins/vcs-git/README.md`: per-plugin purpose, configuration, and any plugin-specific behaviour.

## Non-goals

- License header injection into source files.
- CI enforcement of license compliance (e.g., `licensei`).
- Documentation site or generated API docs.

## Capabilities

**Affected capability surfaces:** none (session / vcs / forge / picker / hook — unchanged)

**Proto changes:** none

### New Capabilities

None — this change adds static files only; no new API or plugin surfaces are introduced.

### Modified Capabilities

None — no existing spec-level requirements change.

## Impact

- **Files added:** `LICENSE`, `README.md` (root), `cmd/swm/README.md`, `sdk/go/README.md`, `plugins/forge-github/README.md`, `plugins/picker-fzf/README.md`, `plugins/session-tmux/README.md`, `plugins/vcs-git/README.md`.
- **Code:** no changes.
- **Dependencies:** none.
- **APIs / proto:** none.
- **CI:** no changes required; the files are inert to the existing `task fmt / lint / test` pipeline.
