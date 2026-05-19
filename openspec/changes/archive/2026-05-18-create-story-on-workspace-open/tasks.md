## 1. Spec update

- [x] 1.1 `openspec/specs/workflow-commands/spec.md` (`cmd/swm`): apply the delta — replace the single "Story not found" scenario with the four new scenarios (user confirms, user declines, non-TTY stdin, pre-story-create hook aborts) and add the interactive-creation prose to the requirement body

## 2. Tests — red phase (cmd/swm)

- [x] 2.1 `cmd/swm`: write failing integration test — `swm workspace open nonexistent` with non-TTY stdin exits non-zero immediately with no prompt (unchanged scripted behavior)
- [x] 2.2 `cmd/swm`: write failing integration test — `swm workspace open nonexistent` with TTY stdin and user answers `n` exits non-zero
- [x] 2.3 `cmd/swm`: write failing integration test — `swm workspace open nonexistent` with TTY stdin and user answers `y`, `pre-story-create` hooks run, story is created, open proceeds
- [x] 2.4 `cmd/swm`: write failing integration test — `swm workspace open nonexistent` with TTY stdin, user answers `y`, a `pre-story-create` hook exits non-zero → story NOT created, command exits non-zero

## 3. Implementation — green phase (cmd/swm)

- [x] 3.1 `cmd/swm`: extract the story-creation logic from `storyCreateCmd` (store.Create + pre/post-story-create hookexec.Run calls) into a shared `createStory(ctx, store, hookExec, name)` helper function
- [x] 3.2 `cmd/swm`: in the workspace-open handler, after `store.Get(name)` returns a "not found" error, add a TTY check via `golang.org/x/term.IsTerminal(int(os.Stdin.Fd()))`; if not a TTY, return the existing error unchanged
- [x] 3.3 `cmd/swm`: write the confirmation prompt (`Story '<name>' does not exist. Create it? [y/N]: `) to `os.Stderr` and read one line from `os.Stdin` via `bufio.NewReader`
- [x] 3.4 `cmd/swm`: on `y`/`Y` answer call `createStory(...)` and continue the open flow on success; on any other answer return the existing "story not found" error

## 4. Verification

- [x] 4.1 `cmd/swm`: run `task fmt` and fix any formatting issues
- [x] 4.2 `cmd/swm`: run `task lint` and fix any lint issues
- [x] 4.3 run `task test` and confirm all tests pass
