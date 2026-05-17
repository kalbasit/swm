## MODIFIED Requirements

### Requirement: Hook executor package
`cmd/swm/internal/hookexec` SHALL provide a `Run(ctx context.Context, cfg RunConfig) error` function that discovers and executes plain executables for a named lifecycle event. `RunConfig` SHALL carry: `event` (string), `codeRoot` (string), `storyName` (string), `projectHost` (string, optional), `projectPath` (string, optional — segments joined by `/`), `worktreePath` (string, optional), `repoPath` (string, optional), `workDir` (string, optional — working directory for hook processes).

#### Scenario: No hooks configured — exits cleanly
- **WHEN** `hookexec.Run` is called and none of the three hook tier directories exist
- **THEN** the function returns nil without executing anything

#### Scenario: Executables run in lexical order within a tier
- **WHEN** a global hook tier contains `00-first` and `10-second`
- **THEN** `00-first` is executed before `10-second`

## ADDED Requirements

### Requirement: Hook working directory
When `RunConfig.WorkDir` is non-empty, the executor SHALL set `cmd.Dir` to `WorkDir` for every hook process it spawns. When `WorkDir` is empty, the hook process SHALL inherit the working directory of the swm process.

#### Scenario: Hook runs in the specified working directory
- **WHEN** `hookexec.Run` is called with `WorkDir` set to a valid directory path
- **THEN** each hook executable runs with that directory as its working directory (i.e. `$PWD` equals `WorkDir`)

#### Scenario: Hook inherits cwd when WorkDir is empty
- **WHEN** `hookexec.Run` is called with an empty `WorkDir`
- **THEN** hook executables inherit the working directory of the swm process

### Requirement: Hook working directory per event
Each call site SHALL populate `WorkDir` according to the event:

- `pre-story-create`, `post-story-create`, `pre-story-remove`, `post-story-remove`: `codeRoot`
- `pre-worktree-create`: `repoPath` (repo exists; worktree not yet created)
- `post-worktree-create`: `worktreePath` (worktree was just created)
- `pre-worktree-remove`: `worktreePath` (last chance to act inside the worktree)
- `post-worktree-remove`: `repoPath` (worktree gone; repo still present)
- `pre-clone`: `codeRoot` (repo does not exist yet)
- `post-clone`: `repoPath` (newly cloned repo)
- `pre-workspace-open`, `post-workspace-open`: `worktreePath`

#### Scenario: post-worktree-create hook runs inside the worktree
- **WHEN** `swm workspace open` creates a worktree and runs `post-worktree-create` hooks
- **THEN** the hook's working directory is the newly created worktree path

#### Scenario: post-worktree-remove hook runs inside the repo
- **WHEN** `swm story remove` removes a worktree and runs `post-worktree-remove` hooks
- **THEN** the hook's working directory is the canonical repository path, allowing operations such as `git worktree prune`

#### Scenario: pre-worktree-remove hook runs inside the worktree
- **WHEN** `swm story remove` is about to remove a worktree and runs `pre-worktree-remove` hooks
- **THEN** the hook's working directory is the worktree path, allowing final cleanup inside it

#### Scenario: story-level hooks run at code root
- **WHEN** `swm story create` or `swm story remove` runs story-level hooks
- **THEN** the hook's working directory is `codeRoot`, since no single repository context applies
