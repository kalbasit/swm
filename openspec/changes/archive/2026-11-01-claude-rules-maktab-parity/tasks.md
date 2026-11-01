# Tasks

## 1. Rule text

- [x] 1.1 In `contrib/claude-rules/swm-story-confinement.md`, replace the "Detect the story"
  paragraph so it uses the story resolved by `swm-story-from-path.md` when installed, falls back
  to `$SWM_STORY`, and defines `$SWM_STORY` as that resolved story. Verify: the paragraph names
  `swm-story-from-path.md` and the fallback, and `grep -ri maktab contrib/` returns nothing.

## 2. Verification

- [x] 2.1 Check the spec contract by hand: both rule files are top-level `.md` files, and the
  README Contrib table has one row per file. Verify: `ls contrib/claude-rules` against
  `grep -c 'contrib/claude-rules/' README.md`.
- [x] 2.2 Run `task fmt`, `task lint` and `task test`, and `openspec validate claude-rules-maktab-parity`.
  Verify: all exit 0 and `task fmt` leaves no diff.

## Workflow follow-up

- Sync the delta spec into `openspec/specs/agent-rules/`, archive the change, open the PR.
