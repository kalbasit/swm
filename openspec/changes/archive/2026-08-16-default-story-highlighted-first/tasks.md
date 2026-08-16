## 1. Tests first (cmd/swm)

- [x] 1.1 In `cmd/swm/internal/cli/workspace/sort_stories_test.go`, rewrite `TestSortStoriesForPicker_DefaultPinnedLast` into `TestSortStoriesForPicker_DefaultPinnedFirst`: with `_default` plus two feature stories, assert `got[0].Name == _default`, then the feature stories follow in CreatedAt-descending order (`new`, then `old`). Confirm it fails against the current implementation (TDD red).
- [x] 1.2 In the same test file, verify the existing `TestSortStoriesForPicker_DescendingByCreatedAt`, `_TiesOrderedLexicographically`, `_OnlyDefault`, and `_DoesNotMutateInput` cases still express intended behavior once `_default` is first (only ordering with `_default` present changes; the no-default cases are unaffected).
- [x] 1.3 Grep `cmd/swm/internal/cli/workspace/` (esp. `open_test.go`, `story_display_test.go`) for any assertion that `_default` appears last in the picker stream; update those expectations to first. (cmd/swm)

## 2. Implementation (cmd/swm)

- [x] 2.1 In `cmd/swm/internal/cli/workspace/sort_stories.go`, invert the `_default` pin in `SortStoriesForPicker`'s comparator so `_default` sorts before non-default stories (`aDefault && !bDefault -> -1`, `!aDefault && bDefault -> 1`); leave the CreatedAt-descending and lexicographic tie-break logic unchanged. Update the function doc comment ("pinned last" → "pinned first, under the picker cursor"). (cmd/swm)
- [x] 2.2 Run the workspace package tests; confirm 1.1 now passes (TDD green) and all other workspace tests pass. (cmd/swm)

## 3. Verify

- [x] 3.1 Run `task fmt`, `task lint`, and `task test`; confirm each exits 0. (cmd/swm)
- [x] 3.2 Confirm no proto files changed (no `proto/` diff), so no vendor-hash regen or plugin rebuild is required.
