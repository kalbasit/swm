## Context

The repository uses GitHub Actions for CI. The existing `ci.yml` workflow has an
aggregate `ci` job that serves as the single required branch-protection status
check. Adding the OpenSpec guard as a new job in that same workflow is the
lowest-friction approach: no new workflow file, no new branch-protection rule,
and consistent trigger semantics.

`openspec/changes/` holds one subdirectory per in-progress change. Completed
changes are moved under `openspec/changes/archive/`. An "active" change is any
direct child directory of `openspec/changes/` that is **not** `archive`.

## Goals / Non-Goals

**Goals**
- Fail CI on any PR that would merge while `openspec/changes/` contains at
  least one non-`archive` entry.
- Zero additional branch-protection configuration required.

**Non-Goals**
- Does not block direct pushes to `main` (only PRs).
- Does not validate content or completeness of OpenSpec artifacts.
- Does not auto-archive or mutate `openspec/changes/`.

## Decisions

### Add a new `openspec-guard` job to `ci.yml`

**Rationale**: The existing `ci` aggregate job is the only required status check.
Adding a new job there and listing it in `ci`'s `needs` is the minimal change.
Creating a separate workflow would require an additional branch-protection rule.

**Alternative considered**: A separate `openspec-guard.yml` workflow. Rejected —
requires a second required status check entry, which adds maintenance overhead
when status check names change.

### Detection via `find` with a directory listing, not file presence

The check runs:
```bash
find openspec/changes -maxdepth 1 -mindepth 1 -type d ! -name archive
```
If the output is non-empty, the job fails.

**Alternative considered**: Checking `git diff --name-only` against base branch.
Rejected — this would only catch changes introduced by the PR, not pre-existing
active changes committed on an earlier branch. The filesystem check is
authoritative regardless of when the change was introduced.

### No `actions/checkout` token requirement

The check is read-only against the checked-out tree. It needs no write token
and no secrets, so it can run on fork PRs unlike the `generate` job.

## Risks / Trade-offs

- [Risk] A developer with a legitimate active change cannot merge an unrelated
  PR until the change is archived. → Mitigation: document the expected lifecycle
  (archive before merging); the OpenSpec `verify` + `archive` flow is fast.

## Migration Plan

1. Add the `openspec-guard` job to `ci.yml`.
2. Add `openspec-guard` to `ci`'s `needs` list and its result-check array.
3. No branch-protection changes required — `ci` already absorbs it.
4. Rollback: revert the two edits to `ci.yml`.

## Open Questions

_(none)_
