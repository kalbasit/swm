## Why

Phase 1 shipped `swm workspace open --story <name>` as a fixed-flow command that
opens a workspace for all currently-attached projects with no interactive selection.
The v1 UX — which users depend on daily — lets you pick a project interactively
using fzf, then lazily creates a worktree if that project isn't yet attached to
the story. Phase 2 restores that interactive picker flow and delivers the `picker-fzf`
plugin that drives it (see TDD §7.4 and TDD §6.3).

## What Changes

- Implement `plugins/picker-fzf`: a fully functional `swm-plugin-picker-fzf` binary
  that wraps `fzf` via bidirectional gRPC streaming (`Pick` RPC).
- Implement `sdk/go/picker` Serve helper (currently a stub with no `GRPCPlugin` wiring).
- Wire the picker into `swm workspace open`: enumerate candidates (attached projects
  + all repositories on disk), stream them to the picker plugin, then open or lazily
  create a pane group for the selected project.
- Add `pane_group_command` config support to `session-tmux` so users can substitute
  laio, smug, or any other layout manager (see TDD §6.3 session capability).
- Extend `hostsvc` with `CallCapability` so the session plugin can delegate VCS
  lookups to the host rather than coupling directly.

## Capabilities

### New Capabilities

- `picker-fzf`: fzf-backed implementation of the `Picker` gRPC service; streams
  candidates in, streams selection out; supports multi-select.

### Modified Capabilities

- `workspace-open`: interactive project selection via picker plugin; lazy worktree
  creation for newly selected projects; falls back to listing all attached projects
  when no picker is configured.
- `session-tmux`: add `pane_group_command` config key; when set, run the command
  (with `{{worktree_path}}`, `{{story_name}}`, `{{project_id}}` template vars) instead
  of the default shell+editor pane layout.

## Impact

- `plugins/picker-fzf/`: new module and binary.
- `sdk/go/picker/`: replace stub `GRPCPlugin` with real wiring (mirrors session/vcs done in Phase 1).
- `cmd/swm/internal/cli/workspace/open.go`: add picker integration and lazy-worktree logic.
- `cmd/swm/internal/hostsvc/server.go`: add `CallCapability` RPC implementation.
- `plugins/session-tmux/internal/session/tmux.go`: add `pane_group_command` support.
- `cmd/swm/internal/pluginmgr/manager.go`: add picker capability to discovery and validation.
- Proto: no new RPCs — `Picker.Pick` and `Host.CallCapability` are already defined in
  `proto/swm/plugin/v1/`; only the host-side `CallCapability` implementation is missing.

## Non-goals

- Implementing any picker plugin other than fzf.
- `swm plugin install` command (Phase 4).
- Hooks for worktree-create events triggered by lazy attachment (Phase 3).
- Forge integration or PR listing in the picker (Phase 3).
