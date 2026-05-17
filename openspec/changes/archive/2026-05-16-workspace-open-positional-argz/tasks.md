## 1. Tests (TDD — write before implementation)

- [x] 1.1 cmd/swm: Add unit tests in `open_test.go` covering positional arg resolves story name, positional arg takes priority over `$SWM_STORY`, and no positional arg falls back to `$SWM_STORY` then default story
- [x] 1.2 cmd/swm: Update integration tests in `cmd/swm/tests/integration/` that call `workspace open --story <name>` to use the positional form `workspace open <name>`

## 2. Core Implementation

- [x] 2.1 cmd/swm: In `NewOpenCmd` (`cmd/swm/internal/cli/workspace/open.go`), remove `StringVar` registration for `--story`, add `Args: cobra.MaximumNArgs(1)` to the command, and extract story name from `args[0]` in `RunE` when present
- [x] 2.2 cmd/swm: Update the `Use` field from `"open"` to `"open [story-name]"` and the `Short` description to reflect positional arg

## 3. Verification

- [x] 3.1 cmd/swm: Run `task fmt && task lint && task test` and confirm all exit 0
