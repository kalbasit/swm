# Derive the swm story from the working directory, not the inherited env

> Copy this file into your agent's rules directory (for Claude Code:
> `~/.claude/rules/swm-story-from-path.md`) next to `swm-story-confinement.md`.

`$SWM_STORY` can be inherited from whatever shell started the Claude Code daemon (background
jobs reuse pre-spawned processes), so it may name a story unrelated to this session. Never
accept it as-is.

**At the start of every session**, before any repo work, resolve the effective story from the
primary working directory (`CODE_ROOT=$(swm config get code_root)`):

- cwd under `$CODE_ROOT/stories/<STORY>/...` → the story is `<STORY>`.
- cwd under `$CODE_ROOT/repositories/...` (the canonical clone, i.e. master) → **no story**,
  even if `$SWM_STORY` is set.
- cwd inside a git worktree (e.g. `.claude/worktrees/...`) → apply the same check to the path
  the worktree lives under: a worktree of a canonical clone is still **no story**; a worktree
  under `stories/<STORY>/` is `<STORY>`.
- Anything else → no story.

Then state it explicitly in your first message — "swm story: cleared (inherited `X`, cwd is the
canonical clone)" or "swm story: set to `<STORY>` (from cwd)" — and use that value, not the env,
for everything the swm-story-confinement rule decides. Since shell state doesn't persist between
Bash calls, prefix commands that depend on it with `unset SWM_STORY;` or
`export SWM_STORY=<STORY>;` as appropriate.

If the resolved story differs from `$SWM_STORY`, mention the mismatch; never move work into the
env's story because of it.

**Why**: a background session started in a canonical clone inherited an unrelated
`SWM_STORY` from the daemon, then fast-forwarded that story's branch and put new work in its
worktree.
