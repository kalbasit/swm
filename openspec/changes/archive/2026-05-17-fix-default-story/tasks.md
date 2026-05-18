## 1. Spec

- [x] 1.1 Sync delta spec `workflow-commands` to main spec (`openspec/specs/workflow-commands/spec.md`) — add the new "Interactive selection with picker — _default story, project not yet attached" scenario and update step 6b of the picker path in `cmd/swm`

## 2. Test (cmd/swm)

- [x] 2.1 Add failing unit test in `cmd/swm/internal/cli/workspace/open_test.go` for the scenario: `_default` story, picker selects unattached project — `vcs.CreateWorktree` must NOT be called and project IS attached to the store

## 3. Implementation (cmd/swm)

- [x] 3.1 In `cmd/swm/internal/cli/workspace/open.go` (`openWithPicker`): wrap the VCS plugin load and `vcs.CreateWorktree` call with `if storyName != cfg.DefaultStory { … }` — hooks and store attachment remain outside the guard

## 4. Verify

- [x] 4.1 Run `task fmt && task lint && task test` and confirm all pass
