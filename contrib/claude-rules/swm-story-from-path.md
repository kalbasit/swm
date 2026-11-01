# Derive the swm story from the working directory, not the inherited env

> Copy this file into your agent's rules directory (for Claude Code:
> `~/.claude/rules/swm-story-from-path.md`) next to `swm-story-confinement.md`.

`$SWM_STORY` can be inherited from whatever shell started the Claude Code daemon (background
jobs reuse pre-spawned processes), so it may name a story unrelated to this session. Never
accept it as-is.

**At the start of every session**, before any repo work, resolve the effective story from the
primary working directory (`CODE_ROOT=$(swm config get code_root)`,
`DEFAULT_STORY=$(swm config get default_story)`):

- cwd under `$CODE_ROOT/stories/<STORY>/...` → the story is `<STORY>`.
- cwd under `$CODE_ROOT/repositories/...` (the canonical clone) → the default story
  (`$DEFAULT_STORY`), even if `$SWM_STORY` names another one.
- cwd inside a git worktree (e.g. `.claude/worktrees/...`) → apply the same check to the path
  the worktree lives under: a worktree of a canonical clone is still the default story; a
  worktree under `stories/<STORY>/` is `<STORY>`.
- Anything else → the default story.

Then state it explicitly in your first message — "swm story: `<DEFAULT_STORY>` (inherited `X`,
cwd is the canonical clone)" or "swm story: `<STORY>` (from cwd)" — and use that value, not the
env, for everything the swm-story-confinement rule decides (including `swm story attach` for the
default story). Since shell state doesn't persist between Bash calls, prefix commands that depend
on it with `export SWM_STORY='<STORY>';`, shell-quoted (single quotes, with any `'` in the name
written as `'\''`) so that shell metacharacters in a story name stay part of the value.

If the resolved story differs from `$SWM_STORY`, mention the mismatch; never move work into the
env's story because of it.

**Why**: a background session started in a canonical clone inherited an unrelated
`SWM_STORY` from the daemon, then fast-forwarded that story's branch and put new work in its
worktree.
