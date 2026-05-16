## Context

Phase 1 shipped a `workspace open` command that opens a workspace for all projects
already attached to a story. The v1 experience users expect is interactive: fzf pops
up, you pick a project, swm lazily creates a worktree if needed and opens a pane
group. Phase 2 delivers that experience through the `Picker` gRPC capability defined
in `proto/swm/plugin/v1/picker.proto`.

The `Pick` RPC is bidirectional streaming (host streams candidates in, plugin streams
selections out), which is the correct model for incremental filtering tools like fzf.
The `sdk/go/picker` package exists as a stub; Phase 2 wires it up the same way
session/vcs were wired in Phase 1.

## Goals / Non-Goals

**Goals:**
- Working `picker-fzf` plugin that wraps fzf with the `Picker.Pick` bidirectional RPC.
- `workspace open` gains interactive project selection when a picker is configured.
- Lazy worktree creation: if the selected project isn't in the story yet, the host
  calls `vcs.CreateWorktree` and updates the story store before opening the pane group.
- `pane_group_command` implemented in session-tmux (it was spec'd in Phase 1 but not coded).
- Picker is optional: without a picker plugin configured, `workspace open` retains the
  Phase 1 behaviour (opens workspace for all attached projects).

**Non-Goals:**
- `host.CallCapability` RPC — session-tmux doesn't need cross-plugin calls for Phase 2;
  the host already derives worktree paths and passes them directly. Deferred to Phase 3.
- Multi-select in the picker (single-select only for `workspace open`).
- Any forge or hook wiring.

## Decisions

### fzf subprocess TTY handling
**Decision:** Run fzf with `/dev/tty` as its stdin and stdout (`os.Stdin = /dev/tty`,
`os.Stdout = /dev/tty`), passing candidates via pipe. The plugin receives `PickItem`
messages from the gRPC stream, accumulates them, then launches fzf with candidates
on stdin; after selection fzf exits and the plugin streams a single `PickResult` back.

**Why:** fzf is a TUI process that requires a real TTY for rendering. Piping through
gRPC without a TTY attachment would produce no output. Attaching `/dev/tty` directly
is the standard approach (used by git, vim, etc.) and works in terminals launched by
swm (where a TTY is always present).

**Alternative considered:** Named pipe bridging with a pty — significantly more complex,
requires a pty library, and adds failure modes. Not worth it for the straightforward
single-selection use case.

### Picker invocation in workspace open
**Decision:** The host accumulates candidates from `hostsvc.ListProjects` (all repos
under the code root), streams them to the picker, receives the selection, then derives
the worktree path and calls `session.OpenPaneGroup`. If the project is not yet attached
to the story, the host inserts the CreateWorktree+story-update steps first.

**Why:** This keeps the picker invocation entirely in the host, which already has access
to the code root, the story store, and both plugins. The plugin never needs to know about
story state or filesystem layout — that knowledge stays in the host.

**Alternative considered:** Passing worktree paths to the session plugin and letting it
decide whether to use a picker. Rejected: session plugins shouldn't drive story/project
logic; that's host domain.

### Picker-absent fallback
**Decision:** When no picker plugin is configured, `workspace open` skips the
interactive step entirely and calls `session.OpenWorkspace` with all attached projects'
worktree paths (the Phase 1 behaviour), exactly as today.

**Why:** Keeps the UX predictable in headless/CI environments and for users who prefer
a simpler workflow. The behaviour degrades gracefully.

### pane_group_command implementation
**Decision:** session-tmux reads `pane_group_command` from its config block (via
`host.GetConfig` already wired in Phase 1). Template variables (`{{worktree_path}}`,
`{{story_name}}`, `{{project_id}}`) are substituted with `strings.ReplaceAll` before
the command is passed as the initial-command to `tmux new-session -c`.

**Why:** The spec (Phase 1 `session-tmux/spec.md`) already describes this behaviour;
Phase 2 is purely implementing it. `strings.ReplaceAll` is simpler than `text/template`
for the constrained variable set and avoids template parse errors from user-supplied
strings with `{{` in other contexts.

## Risks / Trade-offs

- **TTY availability:** If swm is invoked in a context with no TTY (CI, pipe, `swm ...
  | other`), the fzf subprocess will fail to start. Mitigation: detect `os.Stdin` is
  not a TTY and fall back to the no-picker path (returning an error that the host can
  handle gracefully).
- **Large candidate sets:** streaming thousands of `PickItem` messages before launching
  fzf adds latency. Mitigation: fzf supports incremental input; we can pipe directly
  rather than accumulating first, but that requires a more complex goroutine setup.
  For Phase 2, accumulate-then-launch is acceptable; the optimisation can come later.
- **pane_group_command injection:** user-provided command strings run as-is in tmux.
  This is intentional (user controls their own config) and consistent with how tools
  like smug/tmuxinator work.

## Open Questions

None blocking Phase 2. `host.CallCapability` design can wait for Phase 3 when forge
plugins need to call back into the vcs capability.
