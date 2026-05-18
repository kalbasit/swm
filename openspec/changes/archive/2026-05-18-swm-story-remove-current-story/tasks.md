## 1. Spec — Write failing tests first (TDD red phase)

- [x] 1.1 `cmd/swm` — add unit test cases to `remove_test.go`: (a) no arg + `$SWM_STORY` set removes that story, (b) no arg + `$SWM_STORY` unset returns an error, (c) explicit arg overrides `$SWM_STORY`
- [x] 1.2 `cmd/swm` — add integration test cases to `tests/integration/integration_test.go` covering the same three scenarios end-to-end

## 2. Implementation (green phase)

- [x] 2.1 `cmd/swm` — in `NewRemoveCmd` change `cobra.ExactArgs(1)` to `cobra.MaximumNArgs(1)`
- [x] 2.2 `cmd/swm` — in `RunE`, when `len(args) == 0` read `os.Getenv("SWM_STORY")`; if empty return a descriptive error; if set assign to `name`

## 3. Documentation

- [x] 3.1 `cmd/swm` — update `README.md`: change usage line for `swm story remove` from `swm story remove <name> [-f | --force]` to `swm story remove [<name>] [-f | --force]`; add a note that when `<name>` is omitted `$SWM_STORY` is used

## 4. Verify

- [x] 4.1 Run `task fmt` and confirm exit 0
- [x] 4.2 Run `task lint` and confirm exit 0
- [x] 4.3 Run `task test` and confirm all tests pass
