## 1. Add openspec-guard job to ci.yml

- [x] 1.1 Add `openspec-guard` job to `.github/workflows/ci.yml` that checks out the repo and runs `find openspec/changes -maxdepth 1 -mindepth 1 -type d ! -name archive` — fail if output is non-empty
- [x] 1.2 Add `openspec-guard` to the `ci` aggregate job's `needs` list
- [x] 1.3 Add `"${{ needs.openspec-guard.result }}"` to the results array in the `ci` job's status-check script

## 2. Verify the guard works correctly

- [x] 2.1 Confirm the guard job passes on the current branch (no active changes other than `archive`)
- [x] 2.2 Manually verify the check logic: create a dummy directory under `openspec/changes/` locally and confirm `find` returns it; remove the directory
- [ ] 2.3 Open a draft PR and confirm the `ci` check fails when an active change is present, and passes after archiving it
