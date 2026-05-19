## Why

PRs can currently be merged while an OpenSpec change is still active (not
archived), meaning code whose design intent was never finalized or verified can
ship. A CI gate on the OpenSpec change directory closes this gap.

## What Changes

- Add a new GitHub Actions workflow job that inspects `openspec/changes/` and
  fails if any entry other than `openspec/changes/archive` is present.
- The check runs on every pull request targeting `main`.
- The new job is wired into the aggregate `ci` status check so branch
  protection requires no additional configuration.

## Capabilities

### New Capabilities

_(none)_

### Modified Capabilities

- **github-ci-cd** — adds one new requirement: PRs must not have any active
  (non-archived) OpenSpec changes. A delta spec documents this requirement.

## Impact

- `.github/workflows/` — one new workflow file (or a new job in the existing
  CI workflow).
- `openspec/specs/github-ci-cd/spec.md` — no direct edit; a delta spec file
  covers the new requirement.

## Non-goals

- Enforcing internal structure or content quality of OpenSpec artifacts.
- Blocking pushes directly to `main` (only PRs are gated).
- Automated archiving or any mutation of the `openspec/changes/` tree.
