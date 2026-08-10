## Why

An agent or script working inside a story often needs a project's worktree to
exist before it can operate in it, but today the only command that creates one
is `swm workspace open` — which also opens and *switches* a tmux workspace,
disrupting the human's current session. That gap forced a documented workaround
(a Claude rule) where the agent shells out to raw `git worktree add`, bypassing
swm's bookkeeping so the story JSON `projects` list silently drifts out of sync.
We need a first-class, idempotent, session-free way to guarantee a project is
attached to a story.

## What Changes

- Add **`swm story attach [<story-name>]`** — an idempotent command that ensures
  the current directory's project is attached to the resolved story.
  - **Story resolution**: positional `<story-name>` → `$SWM_STORY` → error when
    neither resolves (mirrors `swm workspace open` / `swm story remove`).
  - **Project resolution**: derived from the current working directory's
    canonical repository.
  - **Already attached** (worktree present / project in the story's `projects`):
    exit `0` as a no-op.
  - **Not attached**: run `pre-worktree-create` hooks, call `vcs.CreateWorktree`,
    attach the project in the story store, then run `post-worktree-create` hooks
    — the same sequence as `swm workspace open` steps 6a–6d in the
    `workflow-commands` spec.
  - Performs **no** session/tmux work; the human's current session is never
    touched.
  - For the `default_story`, no worktree is created (the canonical
    `repositories/` path is the checkout), matching existing `_default` handling.
- Add a copyable **`contrib/claude-rules/swm-story-confinement.md`** and update
  its guidance to run `swm story attach` instead of raw `git worktree add`,
  restoring correct bookkeeping.

## Capabilities

### New Capabilities

_(none)_

### Modified Capabilities

- `workflow-commands`: add the `swm story attach` requirement and its scenarios
  (story resolution precedence, project-from-cwd resolution, idempotent no-op
  when already attached, `pre`/`post-worktree-create` hook sequencing and
  abort-on-failure, default-story skip, session never invoked).

## Impact

- **Capability surfaces**: vcs (`CreateWorktree`, existing), hook
  (`pre`/`post-worktree-create`, existing). No session/picker/forge surface.
- **Proto**: none — reuses existing `vcs.CreateWorktree` and `hookexec`; no
  version bump, so the nix vendorHash rule does not fire.
- **Code**: new cobra command in `cmd/swm/internal/cli/story/attach.go`; a
  project-from-cwd resolver; reuse of `core/story` store attach and
  `internal/hookexec`. Wire the command into the `story` group in
  `cli/story`/`cli/root.go`.
- **Docs**: new `contrib/claude-rules/swm-story-confinement.md`; the user's
  private `~/.claude/rules/…` copy is updated from it by hand.
- **Shell completion**: `<story-name>` completes to known story names.

## Non-goals

- No tmux/session creation, foreground or background; no workspace or pane-group
  setup, and no `SwitchTo`.
- No project-picker interaction — the target project is always the cwd's repo.
- No changes to any plugin proto contract (`vcs`, `session`, `hook`, `picker`,
  `forge`).
- No extra `git fetch` or branch reconciliation beyond what `vcs.CreateWorktree`
  already performs.
