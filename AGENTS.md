<!-- project-rules:start -->
# Project Rules
These rules are authoritative for all agents.

## Env Execution
# Environment Execution (Nix & direnv)

This repository uses Nix and `direnv` for ALL tooling. In non-interactive shells (daemon, ACP, CI), direnv hooks do NOT run automatically — you cannot assume the environment is already loaded.

## Mandatory bootstrap — no exceptions

Before running ANY command that is not a plain `git` operation, you MUST run these two lines in order, in the same shell session:

```sh
direnv status | grep -q "Found RC allowed 0" || direnv allow .
unset DIRENV_DIR DIRENV_FILE DIRENV_WATCHES DIRENV_DIFF && eval "$(direnv export bash)"
```

- Line 1: only runs `direnv allow .` when the RC is not already allowed (`Found RC allowed 0` means allowed). Calling `direnv allow .` unconditionally touches the allow file's mtime, which triggers unnecessary environment reloads in interactive shells.
- Line 2: **always** unsets stale `DIRENV_*` state variables first, then re-exports the full environment. This is necessary because Claude Code's tool execution shells inherit partial `DIRENV_*` vars from the parent process, causing `direnv export bash` to believe the environment is current and output nothing — leaving Nix devshell tools (e.g. `task`, `bun`, `go`) absent from `PATH`.

Do NOT skip either step. Do NOT assume direnv is already allowed or already loaded. Do NOT run them in a different order.

## What counts as "any command"

This includes, but is not limited to: `task`, `bun`, `go`, `nix`, `docker`, test runners, linters, formatters, and any project script. When in doubt, run the bootstrap first.

## Prohibited

- Do not install missing tools globally (e.g. `npm install -g bun`, `brew install go`). All dependencies come from the Nix flake.
- Do not use `direnv exec . <command>` as a substitute for the bootstrap — it does not export env vars into the current shell and will silently break chained commands.

## No Commits To Main
# No Direct Commits to Main

Never run `git commit` while on the `main` branch.

All work must happen on a feature branch. Use `gs-create` to create a new stacked branch before committing.

## No Panic Outside Main
# No Panics Outside the Main Package

Production code in non-`main` packages MUST NOT call `panic()`. Return an `error` and let the caller decide how to handle it.

## Rules

- Never call `panic()` in any package other than `main`.
- Never use `log.Fatal*` or `os.Exit` outside `main` either — they bypass error handling just like `panic`.
- If an error is genuinely unrecoverable, return it up the stack. The `main` package is the only place allowed to terminate the process.
- If a function signature cannot return an error (e.g. an `ent` `DefaultFunc` that must return a single value), restructure the code so the failure case cannot occur (e.g. build the value directly from inputs that cannot fail). Do not reach for `panic` as an escape hatch.
- If a legitimate invariant check is needed (e.g. guarding against a programmer bug), prefer returning a sentinel error. Only if there is truly no other option, use `panic` with a `//nolint:forbidigo // reason` comment explaining why no alternative exists.

## Exceptions

- `_test.go` files may panic (e.g. in mock stubs for unused interface methods).
- Generated code (e.g. files under `apps/uar/ent/<db>/` that are produced by `ent generate`) is exempt — do not hand-edit generated files to remove panics.
- Hand-written schema files under `ent/<db>/schema/` are NOT generated code and MUST follow this rule.

## Applies To

All hand-written production Go code in non-`main` packages.

## No Skip Git Hooks
# Never Skip Git Hooks

Never use `--no-verify` with any git command (`git commit`, `git merge`, `git push`, etc.).

If a git hook fails, investigate and fix the underlying issue. Do not bypass hooks to work around failures.

## Nolint With Comments
# Nolint Directives Require Comments

All `//nolint:{rule}` directives used to silence legitimate linter false positives must be accompanied by an explanatory comment.

## Format

```go
//nolint:{rule} // comment explaining why this false positive is legitimate
```

## Rules

- Every `//nolint` directive must have a trailing comment explaining the reason.
- The comment should be concise but clear enough for future readers to understand why the linter rule was suppressed.
- Comments must appear on the same line as the `//nolint` directive.
- Do not use `//nolint` to silence genuine issues — fix the underlying problem instead.

## Examples

```go
//nolint:errcheck // error is logged by defer handler, safe to ignore
result := someFunc() // nolint:errcheck would be wrong here

//nolint:gosec // only validating example input, not processing user data
testData := []byte("example")
```

## Applies To

All production code: Go source files, configuration files, and any other code where linter suppressions are used.

## Tdd Required
# TDD Required

All production code changes (new features, bug fixes, refactors) MUST use TDD.

Invoke the `/tdd` skill immediately. Do not proceed without it.

## Verify Before Completion
# Verify Before Reporting Completion

Before reporting that any task is complete, you MUST run both of the following commands and confirm each exits with zero status:

1. `task fmt` — all code must be correctly formatted
2. `task lint` — all code must be correctly linted
3. `task test` — all tests must pass

## Rules

- Never report a task as complete until both commands have been run and succeeded.
- If `task fmt` fails or produces changes, apply the formatting and re-run to confirm it exits cleanly.
- If `task lint` fails or produces changes, apply the linting and re-run to confirm it exits cleanly.
- If `task test` fails, fix the failing tests before reporting completion.
- All commands must exit with status `0`. Any non-zero exit is a blocker.

## Applies To

All tasks that involve production code changes: new features, bug fixes, refactors, and configuration changes.

<!-- project-rules:end -->
