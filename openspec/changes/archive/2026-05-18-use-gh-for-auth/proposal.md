## Why

Users must manually create and maintain a `~/.github_token` file (or configure `token_path`) to authenticate `forge-github`, duplicating credentials already managed by the GitHub CLI (`gh`). Delegating to `gh auth token` removes this friction and lets swm reuse the credential store that developers already keep current.

## What Changes

- `forge-github` resolves the GitHub token using the following priority order:
  1. Explicit `token_path` in plugin config (unchanged behavior for users who opt in)
  2. `gh auth token` subprocess call (new default when `token_path` is not set)
  3. `~/.github_token` file (retained as last-resort fallback for environments without `gh`)
- The "missing token" error message is updated to mention `gh auth login` as the recommended fix.
- No proto changes.

## Non-goals

- Installing or managing the `gh` CLI on behalf of the user.
- Supporting OAuth device-flow or other interactive auth flows directly in swm.
- Changing auth for any plugin other than `forge-github`.

## Capabilities

### New Capabilities

_(none)_

### Modified Capabilities

- **forge-github** — The token-loading requirement changes: `gh auth token` becomes the default source when `token_path` is not configured. The fallback chain and error messaging are updated. Affects `openspec/specs/forge-github/spec.md` (token loading requirement and "Missing token" scenarios).

## Impact

- `plugins/forge-github/internal/forge/github.go` — `tokenFromConfig` method rewritten to invoke `gh auth token` via `exec.Command` when `token_path` is absent.
- `openspec/specs/forge-github/spec.md` — "forge-github token loading" requirement and its scenarios updated to reflect the new resolution order.
- Runtime dependency: `gh` CLI on `$PATH` (soft — plugin degrades to file fallback if absent).
- No proto changes; no version bump required.
