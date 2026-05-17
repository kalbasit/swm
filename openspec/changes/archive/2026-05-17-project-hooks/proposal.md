## Why

The `hook-executor` spec (`openspec/specs/hook-executor/spec.md`) fully defines the hook discovery and execution contract — including `post-worktree-create`, `pre/post-workspace-open`, and other lifecycle events — but `swm workspace open` and related commands never call `hookexec.Run`. Users who place hook scripts in `.swm/hooks/<event>.d/` (per-repo tier) or in global/per-story locations see no execution at all, making the hook system a no-op.

## What Changes

- Wire `hookexec.Run` into `workspace open` for `pre-workspace-open` and `post-workspace-open` events.
- Wire `hookexec.Run` into worktree creation (project added to a story) for `pre-worktree-create` and `post-worktree-create` events.
- Wire `hookexec.Run` into worktree removal for `pre-worktree-remove` and `post-worktree-remove` events.
- Wire `hookexec.Run` into story create/remove for `pre-story-create`, `post-story-create`, `pre-story-remove`, `post-story-remove` events.
- Wire `hookexec.Run` into VCS clone operations for `pre-clone` and `post-clone` events.
- Add `--log-level debug` logging around hook invocations so users can observe hook execution.

## Capabilities

### New Capabilities

_(none)_

### Modified Capabilities

- `hook-executor`: Verify `hookexec.Run` signature and environment contract remain unchanged; no requirement changes expected, but delta spec will capture any discovered gaps.
- `workflow-commands`: The workspace/story/project commands gain hook invocation at each lifecycle point; requirements for each command must be extended to specify which hook events are fired and when (before/after the core operation).

## Impact

- **Code**: `cmd/swm` — workspace, story, and project subcommands; wherever worktree creation/removal, story creation/removal, clone, and workspace open are orchestrated.
- **APIs**: No gRPC proto changes; hooks remain plain executables (not plugins).
- **Dependencies**: `hookexec` package already exists (per spec); this change only adds call sites.
- **Non-goals**: Changing the hook executor's discovery logic, environment variables, or supported event list. Hooks for VCS operations inside plugins (those stay in the plugin boundary). Sandboxing or timeout enforcement for hook scripts.
- **Capabilities surface**: `hook` (plain executables), `workflow-commands` (CLI lifecycle).
- **Proto changes**: None.
