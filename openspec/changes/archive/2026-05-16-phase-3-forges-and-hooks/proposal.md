## Why

Phase 1 and 2 delivered a working story/worktree CLI with git, tmux, and an fzf picker. Phase 3 closes the remaining gaps in the v2 TDD: forge integration (GitHub PR management) and a hook executor (lifecycle events that let repos/stories run custom scripts without needing a full gRPC plugin).

## What Changes

- **New `forge-github` plugin**: implements the `Forge` gRPC capability for GitHub — `ListPullRequests`, `CreatePullRequest`, `GetPullRequest`. Reads a GitHub token from the path configured in `plugins.config.forge-github.token_path`.
- **New `swm pr list` command**: lists open PRs for projects attached to the current story, using the forge plugin.
- **New `swm pr create` command**: creates a PR for the current project/story, with `--title`, `--body`, and `--draft` flags. Derives the remote URL from the resolver, delegates to the forge plugin.
- **New hook executor in the host**: runs plain executables (not gRPC plugins) for lifecycle events (`pre/post-story-create`, `pre/post-story-remove`, `pre/post-worktree-create`, `pre/post-worktree-remove`, `pre/post-clone`, `pre/post-workspace-open`). Searches global, per-repo, and per-story hook tiers in order. `pre-*` hooks aborting (non-zero) cancel the operation. `post-*` failures are logged but do not roll back.
- **Wire hooks into existing commands**: `story create`, `story remove`, `clone`, `workspace open` all gain pre/post hook execution around their existing logic.
- **SDK forge package**: `sdk/go/forge/plugin.go` — real `GRPCPlugin` struct, `Serve`, `NewClient`, mirroring the pattern from sdk/go/picker and sdk/go/session.

## Capabilities

### New Capabilities

- `forge-github`: GitHub Forge plugin implementing ListPullRequests, CreatePullRequest, GetPullRequest over the Forge gRPC service.
- `hook-executor`: Host-side lifecycle hook runner — discovers and executes plain executables in global/per-repo/per-story hook directories for each named event.
- `pr-commands`: New `swm pr list` and `swm pr create` CLI commands backed by the forge plugin.

### Modified Capabilities

- `workflow-commands`: `story create`, `story remove`, `clone`, and `workspace open` gain pre/post hook invocations around their core logic.

## Impact

- **New module**: `plugins/forge-github` (own `go.mod`, added to `go.work`)
- **Modified modules**: `sdk/go` (add forge package), `cmd/swm` (hook executor, pr commands, forge pluginmgr wiring)
- **Dependencies**: `google/go-github/v67` in `plugins/forge-github`; no new dependencies in `cmd/swm` or `sdk/go`
- **Proto**: `forge.proto` already defined in Phase 0; no proto changes needed
- **Config**: `plugins.forges = ["github"]` list in `config.toml`; `plugins.paths.github` for the binary path; `plugins.config.forge-github.token_path` for the GitHub token
