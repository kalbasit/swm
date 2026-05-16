## Why

Phase 0 produced a buildable monorepo skeleton with proto definitions and SDK stubs, but no working behavior. Phase 1 implements the first complete vertical slice — real git and tmux integration — so the tool is actually useful end-to-end. See TDD §10 Phase 1.

## What Changes

- **`cmd/swm/internal/core/`**: story store (JSON CRUD + flock), config loader (TOML), layout resolver (code-root path composition from ProjectID)
- **`cmd/swm/internal/pluginmgr/`**: plugin discovery (config → XDG → PATH), launch (go-plugin client), handshake, dep-graph validation, capability registry
- **`cmd/swm/internal/hostsvc/`**: host gRPC service implementations (GetConfig, GetCodeRoot, ListProjects, GetCurrentStory, Log) passed as callbacks to launched plugins
- **`cmd/swm/internal/cli/`**: four cobra subcommands — `swm story create`, `swm story remove`, `swm clone`, `swm workspace open`
- **`plugins/vcs-git/`**: full implementation of VCS capability (Clone, ParseRemoteURL, CreateWorktree, RemoveWorktree, DetectProjectAtPath)
- **`plugins/session-tmux/`**: full implementation of Session capability (OpenWorkspace, CloseWorkspace, ListWorkspaces, OpenPaneGroup, SwitchTo, IsInsideWorkspace, CurrentContext)
- **`sdk/go/{session,vcs}/`**: replace ErrNotImplemented stubs with real go-plugin gRPC transport (GRPCPlugin impl, client/server wrappers)

## Capabilities

### New Capabilities

- `story-store`: Story JSON persistence — create, get, list, delete with flock during writes; XDG layout at `$XDG_DATA_HOME/swm/stories/<name>.json`
- `plugin-lifecycle`: Plugin discovery (config → XDG → PATH), launch via go-plugin, handshake validation, dep-graph construction and startup error reporting
- `vcs-git`: git VCS plugin — ParseRemoteURL, Clone to canonical path, CreateWorktree per story, RemoveWorktree, DetectProjectAtPath; host derives all on-disk paths from returned ProjectID
- `session-tmux`: tmux session plugin — socket-per-workspace, session-per-pane-group model; OpenWorkspace, CloseWorkspace, ListWorkspaces, OpenPaneGroup, SwitchTo, IsInsideWorkspace, CurrentContext
- `workflow-commands`: CLI behavior for the four Phase 1 commands — orchestration flows from TDD §7.1–7.5

### Modified Capabilities

(none — proto/SDK interfaces are unchanged)

## Impact

- **Modules touched**: `cmd/swm`, `sdk/go`, `plugins/vcs-git`, `plugins/session-tmux`
- **New deps** (cmd/swm): `pelletier/go-toml/v2`, `github.com/hashicorp/go-plugin`, `github.com/adrg/xdg`, `github.com/gofrs/flock`
- **New deps** (plugins/vcs-git): `github.com/go-git/go-git/v5` or shell-out to system `git`
- **New deps** (plugins/session-tmux): shell-out to system `tmux`
- **SDK change**: `sdk/go/{session,vcs}/plugin.go` replace stub `Serve` with real go-plugin GRPCPlugin registration; this is additive, not breaking
- **No proto changes**: all RPCs already defined in Phase 0; no v1→v2 migration needed

## Non-goals

- Picker integration (Phase 2)
- Forge/hooks (Phase 3)
- `swm plugin install` (Phase 4)
- Host services `CallCapability` and `GetCurrentStory` beyond stub level
