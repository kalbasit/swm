## MODIFIED Requirements

### Requirement: forge-github token loading
The plugin SHALL resolve the GitHub token using the following priority order:
1. If `token_path` is configured (via `host.GetConfig`), read the token from that file. Leading/trailing whitespace SHALL be trimmed. If the file is missing or empty, return a `FailedPrecondition` error immediately — do not fall through to other sources.
2. Run `gh auth token` as a subprocess. If `gh` is on `$PATH` and exits 0, use the trimmed stdout as the token.
3. Read the token from `~/.github_token`. Leading/trailing whitespace SHALL be trimmed.

If all three sources fail, the plugin SHALL return a gRPC `FailedPrecondition` error whose message mentions both `gh auth login` and the `token_path` config option as remediation steps.

The token SHALL be resolved at call time (not at startup).

#### Scenario: Token loaded from configured path
- **WHEN** `token_path = "~/.secrets/gh_token"` is set in config and the file contains a valid token
- **THEN** the plugin uses that token for GitHub API requests

#### Scenario: Configured path takes priority over gh
- **WHEN** `token_path` is set and `gh auth token` would also return a token
- **THEN** the plugin uses the token from the configured file path

#### Scenario: Configured path missing returns error immediately
- **WHEN** `token_path` is configured but the file does not exist
- **THEN** the plugin returns a gRPC `FailedPrecondition` error without attempting `gh` or the default file

#### Scenario: gh auth token used as default
- **WHEN** `token_path` is not configured and `gh` is on `$PATH` and authenticated
- **THEN** the plugin uses the token returned by `gh auth token`

#### Scenario: gh not installed falls through to file
- **WHEN** `token_path` is not configured and `gh` is not on `$PATH`
- **THEN** the plugin attempts to read `~/.github_token`

#### Scenario: gh exits non-zero falls through to file
- **WHEN** `token_path` is not configured and `gh auth token` exits with a non-zero status
- **THEN** the plugin attempts to read `~/.github_token`

#### Scenario: Default file fallback used
- **WHEN** `token_path` is not configured, `gh` is unavailable or unauthenticated, and `~/.github_token` exists
- **THEN** the plugin reads the token from `~/.github_token`

#### Scenario: All sources fail returns actionable error
- **WHEN** `token_path` is not configured, `gh auth token` fails, and `~/.github_token` does not exist or is empty
- **THEN** the plugin returns a gRPC `FailedPrecondition` error that mentions `gh auth login` and the `token_path` config option
