## 1. Refactor token resolution in plugins/forge-github

- [x] 1.1 Add a `ghTokenFn func(ctx context.Context) (string, error)` field to the `GitHub` struct (`plugins/forge-github/internal/forge/github.go`) and wire the default implementation (`ghAuthToken`) in `New` and `NewWithBaseURL`
- [x] 1.2 Implement `ghAuthToken(ctx context.Context) (string, error)`: use `exec.LookPath("gh")` to check availability, then run `exec.CommandContext(ctx, "gh", "auth", "token")` and return trimmed stdout on exit 0, or a sentinel error on failure
- [x] 1.3 Rewrite `tokenFromConfig` to implement the three-step resolution order: (1) explicit `token_path` → read file and return, never fall through; (2) call `ghTokenFn`; (3) read `~/.github_token`; if all fail, return `FailedPrecondition` with an error message mentioning `gh auth login` and the `token_path` config option

## 2. Unit tests in plugins/forge-github

- [x] 2.1 Write table-driven tests for `tokenFromConfig` covering: configured path hit, configured path missing (no fallthrough), `gh` token used as default, `gh` not installed falls through to file, `gh` exits non-zero falls through to file, file fallback used, all sources fail returns actionable error
- [x] 2.2 Wire the injected `ghTokenFn` stub into test cases via `NewWithBaseURL` or a test helper that sets the field directly (table-driven, no subprocess spawned)

## 3. Verify

- [x] 3.1 Run `task fmt` in `plugins/forge-github` and confirm exit 0
- [x] 3.2 Run `task lint` in `plugins/forge-github` and confirm exit 0
- [x] 3.3 Run `task test` and confirm all tests pass
