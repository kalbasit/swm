## Context

Phases 1 and 2 delivered a working story/worktree/workspace CLI backed by git, tmux, and fzf. The proto definitions for `Forge` and `Host.Log` were declared in Phase 0 but never implemented. Phase 3 closes two remaining gaps from the TDD:

1. **Forge integration**: the `forge-github` plugin and `swm pr list/create` CLI commands let users manage GitHub pull requests without leaving the terminal.
2. **Hook executor**: lifecycle hooks let repos and users run plain shell scripts (not gRPC plugins) at key moments (story create/remove, clone, worktree create/remove, workspace open). This replaces v1's `~/.config/swm/hooks/coder/` mechanism with a richer, scoped model.

Existing code to be aware of:
- `cmd/swm/internal/pluginmgr/manager.go`: already handles `session`, `vcs`, `picker` capabilities; will gain `forge` support.
- `cmd/swm/internal/hostsvc/server.go`: already serves `GetConfig`, `GetCodeRoot`, `ListProjects`, `GetCurrentStory`, `Log`, `CallCapability`.
- `proto/swm/plugin/v1/forge.proto`: `Forge` service and messages already defined.
- `plugins/forge-github/`: skeleton module already exists (go.mod + main.go stub).
- `sdk/go/forge/plugin.go`: exists as a stub — needs the same real GRPCPlugin pattern as picker.

## Goals / Non-Goals

**Goals:**
- `forge-github` plugin that connects to the GitHub API for PR operations (list, create, get)
- `sdk/go/forge` real GRPCPlugin wiring (mirrors picker pattern from Phase 2)
- `pluginmgr` forge capability wiring (supports a list of forge plugins, not just one)
- `swm pr list` and `swm pr create` CLI commands
- Hook executor in `cmd/swm/internal/hookexec/` that discovers and runs executables from three tiers: global, per-repo, per-story
- `pre/post-*` hooks wired into `story create`, `story remove`, `clone`, `workspace open`

**Non-Goals:**
- Forge operations beyond PR list/create/get (issues, CI status, releases — future)
- Non-GitHub forges (`forge-gitlab`, `forge-gitea`)
- Hook plugin capability (hooks are plain executables, not gRPC)
- `swm plugin install` (Phase 4)
- PR review workflows (comment, approve, merge)

## Decisions

### D1: Forge plugin config is a list, not a single value

The TDD specifies `forges = ["github"]` as a list because a single story may touch repos on multiple hosts (e.g. `github.com` + `gitlab.com`). The pluginmgr therefore maintains a map of `hostname → ForgeClient`, populated from the forge plugin's `Info().hostnames` field. When a PR command needs a forge for `github.com/kalbasit/swm`, it looks up `github.com` in the map.

**Alternative**: single forge plugin like session/vcs. Rejected — the TDD explicitly calls out multi-forge support and the forge Info RPC already returns claimed hostnames.

### D2: Hook executor is a standalone package, not baked into each command

A `cmd/swm/internal/hookexec` package with a single `Run(ctx, event, env)` function keeps the hook logic isolated and testable. Each command calls `hookexec.Run` before and after its core action. The executor searches the three tiers defined in the TDD (global, per-repo, per-story) and runs files in lexical order within each tier.

**Alternative**: inline hook logic in each command. Rejected — code duplication, hard to test uniformly.

### D3: `pre-*` hooks abort on non-zero; `post-*` hooks log and continue

Consistent with the TDD and the behavior of git hooks. `pre-*` failures stop the operation cleanly; `post-*` failures are reported but do not roll back (rollback is often impossible and user expectations align with "already done").

### D4: Hook environment variables match the TDD exactly

Each hook invocation receives: `SWM_HOOK`, `SWM_STORY`, `SWM_PROJECT_HOST`, `SWM_PROJECT_PATH` (segments joined by `/`), `SWM_WORKTREE_PATH`, `SWM_REPO_PATH`. For events without a project context (e.g. `pre-story-create`), the project-related vars are empty. Stdin carries a JSON document with the same fields.

### D5: `swm pr create` derives the remote URL from git, not from stored state

The forge plugin receives a `ProjectID`. The CLI derives it from the current working directory via `vcs.DetectProjectAtPath` (already implemented in vcs-git). This avoids storing remote URLs in the story JSON and keeps the forge plugin stateless.

### D6: GitHub token path defaults to `~/.github_token` if not configured

Simple, predictable default that matches common convention. Users with more complex setups (keyring, 1Password CLI) can configure `plugins.config.forge-github.token_path` to point at a file wrapping their credential helper. The plugin reads the file at call time (not at startup), so rotation works without restart.

## Risks / Trade-offs

- **GitHub API rate limits**: unauthenticated requests are limited to 60/hour. The plugin always uses a token; if the token file is missing the plugin returns `FailedPrecondition`. Risk is low for normal use.
- **Hook executor ordering**: within a tier, lexical order of filenames. Users must name hooks carefully (e.g. `00-direnv`, `10-npm-install`) to control ordering. This is well-understood (same as git hooks), but slightly surprising to newcomers.
- **Hook stdin JSON pipe**: if a hook doesn't read stdin, the pipe buffer fills and the executor could block. Mitigation: `hookexec` writes stdin in a goroutine and closes it, so the hook process can ignore stdin without blocking.
- **forge plugin list vs single**: the pluginmgr needs to handle multiple forge plugins loaded simultaneously. This slightly complicates `pluginmgr.Close()` (must close all forge clients). Low risk — the pattern is the same as for other capabilities.

## Open Questions

- Should `swm pr create` open `$EDITOR` for the body if `--body` is not supplied? Deferred: the initial implementation requires `--body` or defaults to empty.
- Should hook timeouts be configurable? Deferred: initial implementation has no timeout; a future `[hooks] timeout_seconds` config can be added.
