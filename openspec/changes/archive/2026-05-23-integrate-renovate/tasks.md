## 1. Prerequisite Check

- [x] 1.1 Verify that the Mend Renovate GitHub App is installed on the `kalbasit` GitHub organization (check https://github.com/organizations/kalbasit/settings/installations); if not, install it and grant access to the `swm` repository
- [x] 1.2 Ensure GitHub labels `dependencies` and `renovate` exist on the `kalbasit/swm` repository; create them if missing (any color is fine)

## 2. Renovate Configuration

- [x] 2.1 Create `renovate.json` at the repository root extending `config:recommended`, with `automerge: false` and the weekly-Monday schedule applied globally
- [x] 2.2 Add a `packageRules` entry enabling the `gomod` manager with explicit `fileMatch` patterns covering all 7 `go.mod` files and setting `groupName: null` (one PR per module)
- [x] 2.3 Add a `packageRules` entry enabling the `nix` manager for `flake.nix` flake input updates with `groupName: null` (one PR per input)
- [x] 2.4 Add a `packageRules` entry enabling the `github-actions` manager for all files under `.github/workflows/`
- [x] 2.5 Add a global `labels` field with `["dependencies", "renovate"]` so all Renovate PRs are labeled

## 3. Validation

- [x] 3.1 Run `npx --yes renovate-config-validator renovate.json` and confirm it exits with no errors
- [x] 3.2 Push the branch and open a draft PR; verify Renovate's onboarding check (or existing scan) detects `renovate.json` and lists all expected dependency sources in its log/dashboard
- [x] 3.3 Confirm no existing PRs or CI steps are broken by the new config (Renovate is read-only until it opens its first scheduled PR)
