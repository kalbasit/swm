## 1. Age Formatting Utility (cmd/swm)

- [x] 1.1 Write failing table-driven tests in `cmd/swm/internal/ageformat/ageformat_test.go` covering all threshold scenarios from the spec: sub-hour rounds up to minutes, sub-day to hours, sub-week to days, sub-month to weeks, sub-year to months, ≥1 year to years; verify exact boundary values
- [x] 1.2 Implement `cmd/swm/internal/ageformat/ageformat.go` — `FormatAge(t time.Time, now time.Time) string` — using rounded-up single-unit logic (minutes → hours → days → weeks → months → years) until all tests in 1.1 pass

## 2. Terminal Width Detection (cmd/swm)

- [x] 2.1 Write failing tests in `cmd/swm/internal/termwidth/termwidth_test.go` for: `$COLUMNS` set to a valid int (returns that value), `$COLUMNS` set to an invalid value (falls through to default), `$COLUMNS` unset (returns 120 default); skip `/dev/tty` tests in unit test suite (covered by manual/integration)
- [x] 2.2 Implement `cmd/swm/internal/termwidth/termwidth.go` — `Detect() int` — with fallback chain: `term.GetSize` on `/dev/tty` (clamp ≤ 0 to next fallback) → `$COLUMNS` env var → 120 default; pass all tests in 2.1

## 3. Story Display String Builder (cmd/swm)

- [x] 3.1 Write failing table-driven tests in `cmd/swm/internal/cli/workspace/story_display_test.go` (or alongside open.go) covering: branch name omitted when equal to story name; branch name shown in parens when different; `_default` always shows as `_default (main repo)`; projects joined with ` · `; projects list trimmed with ` …` when line exceeds width; branch truncated with `…` when projects-trimmed line still exceeds width; zero-project stories show no project suffix; minimum story name always preserved
- [x] 3.2 Implement `buildStoryDisplay(s *story.Story, width int, now time.Time) string` (internal to `cmd/swm/internal/cli/workspace`) applying truncation priority: projects → branch name → story name; pass all tests in 3.1

## 4. Story Sorting (cmd/swm)

- [x] 4.1 Write failing tests for `sortStoriesForPicker(stories []*story.Story) []*story.Story`: feature stories ordered by `CreatedAt` descending; `_default` pinned last regardless of its `CreatedAt`; ties in `CreatedAt` ordered lexicographically by name
- [x] 4.2 Implement `sortStoriesForPicker` until all tests in 4.1 pass

## 5. Story Picker Integration in workspace open (cmd/swm)

- [x] 5.1 Write failing unit tests for the story resolution logic in `cmd/swm/internal/cli/workspace/open_test.go`: no arg + no env + picker available → `pickStory` called; positional arg present → `pickStory` NOT called; `$SWM_STORY` set → `pickStory` NOT called; picker returns `Aborted` → command exits 0 with no workspace opened; picker returns `FailedPrecondition` → falls back to `_default`
- [x] 5.2 Extract `pickStory(ctx context.Context, st store.Store, pickerClient pluginv1.PickerClient, width int) (*story.Story, error)` function in `cmd/swm/internal/cli/workspace/open.go`: calls `store.List`, sorts via `sortStoriesForPicker`, streams `PickItem` entries built via `buildStoryDisplay`, returns selected story or error
- [x] 5.3 Modify the `RunE` handler in `open.go`: when story name is not resolved from positional arg or `$SWM_STORY`, and picker client is loaded, call `pickStory`; on `Aborted` → return nil; on `FailedPrecondition` → fall through to default story; on success → use selected story for the existing project-picker path; pass all tests in 5.1

## 6. Integration Tests (cmd/swm)

- [x] 6.1 Add integration test: `swm workspace open` with no arg and no `$SWM_STORY` streams all stories (including `_default`) to the picker plugin and opens the selected story's workspace
- [x] 6.2 Add integration test: `swm workspace open` with `$SWM_STORY=feat-x` does NOT stream story candidates to picker; proceeds directly to project picker for `feat-x`
- [x] 6.3 Add integration test: `swm workspace open feat-x` (positional arg) does NOT stream story candidates to picker; proceeds directly to project picker for `feat-x`
- [x] 6.4 Add integration test: story picker `PickItem` display strings do not exceed the detected terminal width (use `$COLUMNS=80` env var in test environment)
