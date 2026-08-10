# swm story confinement (git worktrees)

> Copy this file into your agent's rules directory (for Claude Code:
> `~/.claude/rules/swm-story-confinement.md`) to keep an agent confined to a
> single swm story.

The user manages repos with `swm` (https://github.com/kalbasit/swm). Plain clones live at
`~/code/repositories/{hostname}/{name}`. Multi-hour/multi-day work happens in a "story"
worktree instead, opened via `swm workspace open <STORY>`, which lives at
`~/code/stories/{STORY}/{hostname}/{name}` and runs in its own tmux session. Additional repos
get attached to the same story via `swm workspace open --kill-pane` from within it.

**Detect the story**: check `$SWM_STORY` in the environment.

**When `$SWM_STORY` is set**, treat the session as scoped to that story and never break out of it:

- Any repo you touch for this task — even ones not yet part of the story — MUST be operated on
  at `~/code/stories/$SWM_STORY/{hostname}/{name}`, never at
  `~/code/repositories/{hostname}/{name}`.
- If that story worktree doesn't exist yet, create it with swm so the worktree, its hooks, and
  swm's bookkeeping all stay in sync:
  1. `cd` into the repo's canonical clone at `~/code/repositories/{hostname}/{name}`.
  2. Run `swm story attach "$SWM_STORY"`.

  `swm story attach` is idempotent and safe to call blindly: if the worktree already exists it
  does nothing, otherwise it creates the worktree, runs the `pre`/`post-worktree-create` hooks,
  and records the project in the story JSON. It does **not** touch your tmux session — no new
  pane or window is opened — so it is safe to run from an automated agent. Prefer it over a raw
  `git worktree add`: the manual command skips swm's hooks and leaves the story JSON's
  `projects` list out of sync. Only reach for `swm workspace open --kill-pane` when you actually
  want a new tmux pane/session for the repo.

  After attaching, operate on the repo at `~/code/stories/$SWM_STORY/{hostname}/{name}`.

- Before creating or checking out a branch in a story repo, read
  `~/.local/share/swm/stories/$SWM_STORY.json` and use its `branch_name` field verbatim. Do not
  invent a different branch name — all repos attached to the same story share that one branch
  name. (`swm story attach` already checks out this branch for you.)
- That JSON's `projects` array lists the `host`/`segments` of repos already attached to the
  story — cross-check against it before assuming a repo is or isn't part of the current story.

**When `$SWM_STORY` is unset**, this is a quick/ad-hoc session — operate in the current repo
normally, under `~/code/repositories/...`.
