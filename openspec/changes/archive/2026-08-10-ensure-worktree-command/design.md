## Context

Agents and scripts that work inside a story need a project's worktree to exist
before they can operate in it. Today the only command that creates a worktree is
`swm workspace open`, whose flow (see `workflow-commands` spec, `swm workspace
open` steps 6a–6d) creates the worktree, attaches the project to the story store,
and fires `pre`/`post-worktree-create` hooks — but then also calls
`session.OpenWorkspace`, `OpenPaneGroup`, and `SwitchTo`, redirecting the
human's tmux client. Because there was no session-free path, a Claude rule told
agents to run raw `git worktree add`, which skips the story-store attach and
leaves the story JSON `projects` list out of sync.

The worktree-create machinery already exists and is reusable:

- `layout.Resolver.CanonicalPath` / `WorktreePath` compute the paths.
- `pluginv1.VCSClient.CreateWorktree` creates the worktree (idempotent parent
  creation, branch auto-create — see `vcs-git` spec).
- `pluginv1.VCSClient.DetectProjectAtPath` resolves a `ProjectID` from a working
  directory in a VCS-agnostic way (`git -C <path> remote get-url origin` →
  `ParseRemoteURL`), and is already in the proto/`VCSClient` but not yet used by
  the host.
- `core/story` store `Update` attaches a project (flock-guarded) and returns
  `ErrProjectAlreadyAttached` on a duplicate.
- `internal/hookexec` runs the worktree hooks.

## Goals / Non-Goals

**Goals:**

- Add `swm story attach [<story-name>]` — idempotent, safe to call blindly.
- Reuse the exact worktree-create + hook sequence from `workspace open` (6a–6d).
- Resolve the target project from the current working directory, VCS-agnostically.
- Restore correct bookkeeping: every attach updates the story JSON.
- Ship a copyable `contrib/claude-rules/swm-story-confinement.md` that points
  agents at the new command.

**Non-Goals:**

- Any session/tmux interaction (foreground or background). The command never
  loads the session plugin.
- Project-picker interaction; the project is always the cwd's repo.
- Any proto change or new plugin RPC.
- Branch reconciliation beyond what `CreateWorktree` already does.

## Decisions

### 1. Command: `swm story attach [<story-name>]` under the `story` group

Placed in `cmd/swm/internal/cli/story/attach.go`, wired into the existing `story`
cobra group alongside `create`/`remove`/`list`/`branch`. Story-name completion
reuses the same completion helper as `story remove`.

**Why `story attach` over `worktree ensure`/`workspace ensure`:** "attach" names
the durable outcome (a project is part of a story) without leaking the word
"worktree", which is a git-specific mechanism. A future Mercurial/other VCS
plugin may realize the same concept differently; the host contract is "the
project is attached and its working copy exists".

### 2. Story resolution: arg → `$SWM_STORY` → error

Mirrors `swm story remove` and `swm workspace open`: an explicit positional
argument always wins; otherwise fall back to `$SWM_STORY`; if neither resolves,
exit non-zero before doing any work. Unlike `workspace open`, there is **no**
picker fallback — a blind agent call must be deterministic.

If the resolved story does not exist in the store, the command errors (no
interactive create prompt — this must work non-interactively). Story creation
remains `swm story create` / `swm workspace open`.

### 3. Project resolution: `vcs.DetectProjectAtPath(cwd)`

The host resolves the `ProjectID` by calling `vcs.DetectProjectAtPath` with the
current working directory, then composes `RepoPath = CanonicalPath(pid)` and
`WorktreePath = WorktreePath(story, pid)` via the resolver.

**Why not `layout.Resolver.ProjectIDFromPath(cwd)`:** that helper assumes the
path is exactly a repo root under `repositories/` and treats every trailing path
component as a project segment — it breaks when invoked from a subdirectory
(e.g. `.../swm/cmd/swm`) and does not handle a cwd inside a `stories/<x>/…`
worktree. `DetectProjectAtPath` delegates to the VCS plugin, which walks git's
own repo discovery, so it works from any subdirectory and from both the
canonical clone and any existing worktree, and it keeps the host VCS-agnostic
(consistent with the `attach` naming rationale). Its cost is one plugin round
trip, which is negligible for this command.

### 4. Idempotency with three states

After resolving `pid` and loading the story:

1. **Already attached** (`pid` present in `story.Projects`): no-op, exit 0. No
   hooks, no VCS call.
2. **Not attached, worktree absent**: run `pre-worktree-create` hooks → (unless
   default story) `vcs.CreateWorktree` → append to `story.Projects` +
   `store.Update` → run `post-worktree-create` hooks. Identical to `workspace
   open` 6a–6d.
3. **Not attached, worktree already present on disk** (the drift the old rule
   created): reconcile — attach to the store only; skip `CreateWorktree` and the
   create hooks, since nothing is being created. Log that it reconciled existing
   bookkeeping.

A concurrent attach that wins the race surfaces as `ErrProjectAlreadyAttached`
from `store.Update`; the command treats that as success (idempotent), not an
error.

### 5. Default-story handling

When the resolved story is the configured `default_story`, `CreateWorktree` is
skipped (the canonical `repositories/` path is the checkout), matching the
existing `_default` branch in `workspace open`. The project is still recorded in
the store.

### 6. Contrib rule file

`contrib/claude-rules/swm-story-confinement.md` is added as a copyable artifact.
Its worktree-bootstrap section is rewritten to instruct agents to run
`swm story attach` (which creates the worktree, fires hooks, and updates
bookkeeping) instead of raw `git worktree add`. The user copies it to their
private `~/.claude/rules/…` by hand.

## Risks / Trade-offs

- **cwd is not a git repo / no `origin` remote** → `DetectProjectAtPath` returns
  `NotFound`. Mitigation: surface a clear "run this from inside a repository"
  error and exit non-zero.
- **Reconcile path skips hooks** (state 3) → a `post-worktree-create` side effect
  (e.g. `direnv allow`) may be missing for worktrees created by the old manual
  rule. Mitigation: documented; the worktree already existed, so re-running
  create hooks would misrepresent "created". Users wanting hooks can remove and
  re-attach.
- **cwd inside a different story's worktree** → `DetectProjectAtPath` still
  returns the correct `pid`, and the command attaches to the *resolved* story
  (arg/`$SWM_STORY`), which is the intended blind-call behavior.
- **Worktree path exists but is not a valid worktree** → state 3 is entered only
  when a `.git` entry is present at `WorktreePath`, so a stale plain file or an
  unrelated directory does *not* trigger reconciliation (it falls through to the
  create path, where `CreateWorktree` will fail loudly if the path is occupied).
  The narrow residual case — a directory carrying an unrelated `.git` — is an
  accepted limitation; full VCS-aware validation (e.g. `DetectProjectAtPath` on
  the worktree) is left as an implementation detail (see Open Questions).
- **Concurrent `CreateWorktree` race** → two attaches can both pass the existence
  check and run the pre-create hook; the loser's `CreateWorktree` fails. Mitigation:
  on a `CreateWorktree` error, if a worktree is now present at `WorktreePath`, the
  command reconciles the store instead of surfacing the error (idempotent), and
  only a genuine failure with no worktree propagates.

## Migration Plan

Purely additive: a new command plus a new contrib doc. No schema or proto
changes, so the nix `vendorHash` rule does not fire and there is nothing to roll
back beyond removing the command. Adoption is opt-in: users copy the refreshed
contrib rule over their private confinement rule.

## Open Questions

- Should state 3 (reconcile) validate that the existing worktree's checked-out
  branch equals `story.BranchName`, and warn on mismatch? Leaning: warn only, do
  not fail.
- Should reconcile detection key on the store (`projects`) alone, on the
  on-disk worktree, or on `vcs`-reported worktree registration? Leaning: store is
  authoritative for "attached"; a filesystem stat guards `CreateWorktree`.
