## ADDED Requirements

### Requirement: Hook executor package
`cmd/swm/internal/hookexec` SHALL provide a `Run(ctx context.Context, cfg RunConfig) error` function that discovers and executes plain executables for a named lifecycle event. `RunConfig` SHALL carry: `event` (string), `codeRoot` (string), `storyName` (string), `projectHost` (string, optional), `projectPath` (string, optional — segments joined by `/`), `worktreePath` (string, optional), `repoPath` (string, optional).

#### Scenario: No hooks configured — exits cleanly
- **WHEN** `hookexec.Run` is called and none of the three hook tier directories exist
- **THEN** the function returns nil without executing anything

#### Scenario: Executables run in lexical order within a tier
- **WHEN** a global hook tier contains `00-first` and `10-second`
- **THEN** `00-first` is executed before `10-second`

### Requirement: Hook tier discovery
The executor SHALL search three tiers in order for each event `<event>`:
1. **Global**: `$XDG_CONFIG_HOME/swm/hooks/<event>.d/` (all executable files, lexical order)
2. **Per-repo**: `<codeRoot>/repositories/<projectHost>/<projectPath>/.swm/hooks/<event>.d/` (skipped when `projectHost` is empty)
3. **Per-story**: `$XDG_CONFIG_HOME/swm/stories/<storyName>/hooks/<event>.d/` (all executable files, lexical order)

All three tiers SHALL run for every invocation (tiers do not override each other), unless a `pre-*` hook in an earlier tier returns non-zero (which aborts all further execution).

#### Scenario: All three tiers run for post-* event
- **WHEN** `hookexec.Run` is called for event `post-clone` and hooks exist in all three tiers
- **THEN** all three tiers execute in order: global, per-repo, per-story

#### Scenario: Per-repo tier skipped when no project context
- **WHEN** `hookexec.Run` is called with empty `projectHost`
- **THEN** the per-repo tier is skipped entirely

### Requirement: pre-* hooks abort on failure
For events named `pre-*`, if any hook executable exits non-zero, `hookexec.Run` SHALL immediately return an error describing which hook failed and its exit code. No further hooks (within the same tier or later tiers) SHALL be executed.

#### Scenario: pre-* hook fails — operation aborted
- **WHEN** a `pre-worktree-create` hook exits with code 1
- **THEN** `hookexec.Run` returns a non-nil error and no subsequent hooks are run

#### Scenario: pre-* hook succeeds — execution continues
- **WHEN** all `pre-worktree-create` hooks exit 0
- **THEN** `hookexec.Run` returns nil and the calling command proceeds

### Requirement: post-* hooks log failures but continue
For events named `post-*`, if a hook executable exits non-zero, `hookexec.Run` SHALL log the failure (hook path and exit code) but continue executing remaining hooks and return nil.

#### Scenario: post-* hook fails — logged, execution continues
- **WHEN** a `post-worktree-create` hook exits with code 1
- **THEN** the failure is logged, remaining hooks continue executing, and `hookexec.Run` returns nil

### Requirement: Hook environment variables
Each hook executable SHALL be invoked with the following environment variables in addition to the calling process's environment:
- `SWM_HOOK`: the event name (e.g. `pre-story-create`)
- `SWM_STORY`: the story name
- `SWM_PROJECT_HOST`: the project host (e.g. `github.com`), empty if no project context
- `SWM_PROJECT_PATH`: the project path segments joined by `/` (e.g. `kalbasit/swm`), empty if no project context
- `SWM_WORKTREE_PATH`: the full worktree path, empty if not applicable
- `SWM_REPO_PATH`: the full canonical repo path, empty if not applicable

#### Scenario: Environment variables set correctly
- **WHEN** a hook is invoked for `pre-worktree-create` with project `github.com/kalbasit/swm`
- **THEN** the hook process sees `SWM_HOOK=pre-worktree-create`, `SWM_PROJECT_HOST=github.com`, `SWM_PROJECT_PATH=kalbasit/swm`

### Requirement: Hook stdin JSON
Each hook executable SHALL receive a JSON object on stdin with the same fields as the environment variables: `hook`, `story`, `project_host`, `project_path`, `worktree_path`, `repo_path`. The executor SHALL write stdin in a goroutine and close it so hooks that do not read stdin do not block.

#### Scenario: Hook reads stdin JSON
- **WHEN** a hook executable reads its stdin
- **THEN** it receives a valid JSON object with all applicable fields

#### Scenario: Hook ignores stdin — no hang
- **WHEN** a hook executable does not read stdin
- **THEN** the executor does not block; the hook runs to completion normally

### Requirement: Hook supported events
The executor SHALL support the following event names: `pre-story-create`, `post-story-create`, `pre-story-remove`, `post-story-remove`, `pre-worktree-create`, `post-worktree-create`, `pre-worktree-remove`, `post-worktree-remove`, `pre-clone`, `post-clone`, `pre-workspace-open`, `post-workspace-open`.

#### Scenario: All defined events are supported
- **WHEN** `hookexec.Run` is called with any of the defined event names
- **THEN** the function discovers and executes hooks for that event without error (even if no hooks exist)
