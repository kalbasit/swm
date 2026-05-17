## 1. Spec delta

- [x] 1.1 Apply `openspec/changes/story-list/specs/workflow-commands/spec.md` delta to `openspec/specs/workflow-commands/spec.md` (cmd/swm)

## 2. Tests (TDD — write first)

- [x] 2.1 Add table-driven tests for `NewListCmd` in `cmd/swm/internal/cli/story/list_test.go` covering: single story (_default only), multiple stories in lexical order, store error

## 3. Implementation

- [x] 3.1 Create `cmd/swm/internal/cli/story/list.go` implementing `NewListCmd(store coreStory.Store) *cobra.Command` (cmd/swm)
- [x] 3.2 Register `story.NewListCmd(store)` in `storyGroup` inside `cmd/swm/internal/cli/root.go` (cmd/swm)

## 4. Verification

- [x] 4.1 Run `task fmt` and confirm exit 0 (cmd/swm)
- [x] 4.2 Run `task lint` and confirm exit 0 (cmd/swm)
- [x] 4.3 Run `task test` and confirm exit 0 (cmd/swm)
