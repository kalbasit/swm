## Context

`forge-github` currently authenticates by reading a token from a file (`~/.github_token` by default, or a path in plugin config). This requires users to create and maintain a separate credential file even when the GitHub CLI (`gh`) is already installed and authenticated on the same machine.

The change introduces `gh auth token` as the default credential source, falling back to the file-based approach for users without `gh` or who prefer an explicit override.

## Goals / Non-Goals

**Goals:**
- Make `gh auth token` the default token source when no `token_path` is configured
- Retain `token_path` config as an explicit override (takes priority over `gh`)
- Retain `~/.github_token` as a last-resort file fallback (for environments without `gh`)
- Surface a clear, actionable error when all sources fail

**Non-Goals:**
- Installing or managing `gh` on behalf of the user
- Supporting interactive OAuth device-flow within swm
- Changing auth for any plugin other than `forge-github`
- Parsing `gh` config files directly (fragile, tied to `gh` internals)

## Decisions

### Decision 1: Invoke `gh` via subprocess (`exec.Command`)

**Chosen**: `exec.Command("gh", "auth", "token")` — run the `gh` CLI and capture stdout.

**Alternatives considered**:
- Parse `~/.config/gh/hosts.yml` directly — rejected: ties us to `gh` internal format, breaks across `gh` versions, and is fragile with keychain-backed storage.
- Use `go-gh` library — rejected: adds a significant dependency for a one-liner subprocess call; `gh auth token` is the stable public interface.

**Rationale**: `gh auth token` is `gh`'s documented, stable interface for emitting the active token. A subprocess call is a single line of code and picks up any future `gh` keychain/OAuth changes for free.

### Decision 2: Resolution order — config override first, then `gh`, then file fallback

**Chosen**:
1. `token_path` in plugin config → read file at that path (unchanged behavior)
2. `gh auth token` subprocess → use if `gh` is on `$PATH` and exits 0
3. `~/.github_token` file → last resort; maintains backwards compatibility

**Rationale**: Explicit configuration always wins (principle of least surprise). `gh` is the new default for unconfigured installations. The file fallback preserves compatibility for CI environments and users who set up the file before `gh` existed.

### Decision 3: Soft dependency on `gh` — no hard error if absent

If `exec.LookPath("gh")` fails or `gh auth token` exits non-zero, silently fall through to the file fallback. Only emit a `FailedPrecondition` error if all three sources fail, with a message that mentions both `gh auth login` and the `token_path` config option.

**Rationale**: Avoids breaking existing users who do not have `gh` installed.

### Decision 4: Testability via injected executor

Extract the subprocess call behind a `func(ctx context.Context) (string, error)` field on `GitHub` (defaulting to `ghAuthToken`). Tests inject a stub that returns a controlled token without spawning a real process.

**Rationale**: Avoids `exec.Command` in unit tests while keeping the production path simple.

## Risks / Trade-offs

- **`gh` not on `$PATH`** → falls through to file fallback; no regression for existing users.
- **`gh` authenticated to a different account** → token belongs to whichever account `gh` is logged in as. Same risk exists today with the file. Documented as user responsibility.
- **Subprocess overhead** → `gh auth token` is fast (reads a local file/keychain). Called only at RPC time, same as the file read today.
- **Test isolation** → injected executor function prevents subprocess calls in unit tests; integration tests can set `FORGE_GITHUB_API_URL` against a fake server as today.

## Migration Plan

- Additive change: no config migration needed.
- Users with `token_path` configured see no behaviour change.
- Users without any config now get credentials from `gh` automatically (if installed), or fall back to `~/.github_token` as before.
- No rollback complexity — if a release is reverted, the file fallback still works.

## Open Questions

_(none)_
