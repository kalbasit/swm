## 1. Config — add Story section with BranchNameTemplate

- [x] 1.1 `cmd/swm` — Add `Story` struct with `BranchNameTemplate string \`toml:"branch_name_template"\`` field and embed it in `Config` as `Story Story \`toml:"story"\``; update `Defaults()` to set `BranchNameTemplate: "feat/{{.Name}}"`.
- [x] 1.2 `cmd/swm` — Add config tests: missing `[story]` section produces default `"feat/{{.Name}}"`, explicit value is parsed correctly, empty string in TOML falls through to default via `Defaults()` merge.

## 2. Template evaluation helper (TDD)

- [x] 2.1 `cmd/swm` — Write failing tests in `cmd/swm/internal/cli/story/` for a `branchFromTemplate(tpl, name string) (string, error)` helper: (a) valid template `"feat/{{.Name}}"` + `"feat-x"` → `"feat/feat-x"`, (b) custom template `"fix/{{.Name}}"` + `"my-bug"` → `"fix/my-bug"`, (c) invalid template syntax → error, (d) template producing empty string → error.
- [x] 2.2 `cmd/swm` — Implement `branchFromTemplate` (unexported) in `cmd/swm/internal/cli/story/create.go` or a new `branch.go` file; make all tests from 2.1 pass.

## 3. Wire template into swm story create (TDD)

- [x] 3.1 `cmd/swm` — Extend `NewCreateCmd` signature to accept `branchNameTemplate string` (raw template string); update call site in `root.go` to pass `cfg.Story.BranchNameTemplate`.
- [x] 3.2 `cmd/swm` — Update `create_test.go`: (a) no `--branch` + default template → `"feat/<name>"`, (b) no `--branch` + custom template → derived name, (c) `--branch` flag takes precedence over template, (d) invalid template → non-zero exit before hooks run, (e) template producing empty string → non-zero exit before hooks run.
- [x] 3.3 `cmd/swm` — Replace `branch = "feat/" + name` in `RunE` with `branch, err = branchFromTemplate(branchNameTemplate, name)`; return the error immediately (before hooks) on failure; make all tests from 3.2 pass.
- [x] 3.4 `cmd/swm` — Update `--branch` flag description to read `"branch name (default: derived from config branch_name_template)"`.

## 4. Verification

- [x] 4.1 Run `task fmt` and fix any formatting issues.
- [x] 4.2 Run `task lint` and fix any lint issues.
- [x] 4.3 Run `task test` and confirm all tests pass (unit + integration).
