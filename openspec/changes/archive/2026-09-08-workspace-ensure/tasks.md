## 1. Separate making a workspace from going to it

- [x] 1.1 Extract from `openAllAttached` the part that opens the workspace and a pane group, returning their ids, leaving `SwitchTo`, exec and `--kill-pane` in `open.go`; verify `swm workspace open`'s existing tests pass with no edits to them
- [x] 1.2 Let the extracted function take which projects to open pane groups for, so `ensure` can pass every attached project while `open` keeps passing only the first; verify `open` opens exactly one group for a three-project story, because the proposal says `open` is unchanged and opening three would change it

## 2. The command

- [x] 2.1 Add `swm workspace ensure [<story-name>] [--json]` resolving the story from argument, `$SWM_STORY`, then `default_story`; verify each precedence step with a test, and that the picker plugin is never called even when configured with a terminal attached
- [x] 2.2 Refuse an unknown story non-zero, naming it, before any plugin call, and never create one; verify no story file is written and no prompt is read
- [x] 2.3 Print exactly the workspace id and a newline **to stdout**; verify by running the built binary and capturing stdout alone, because cobra's Println goes to stderr and no unit test in this package can tell the two apart once SetOut is called
- [x] 2.4 Add `--json` printing one object with the workspace id and the pane groups opened; verify it parses and carries a group per attached project
- [x] 2.5 Run `pre-workspace-open` and `post-workspace-open` hooks with the same abort/log asymmetry as `open`; verify a failing pre-hook opens nothing and a failing post-hook still exits zero
- [x] 2.6 Verify the command never calls `session.SwitchTo` and never execs, with a terminal attached — this is the property the whole change exists for, so it deserves its own test rather than being implied by another

## 3. Idempotence and the empty case

- [x] 3.1 Verify a second `ensure` against a live workspace exits zero, prints the same id, and opens nothing new
- [x] 3.2 Verify a story with no attached projects yields a workspace, no pane group, and a printed id

## 4. Surface and documentation

- [x] 4.1 Register story-name completion for the positional argument; verify Tab offers story names and not filenames, and that a store error offers nothing rather than erroring
- [x] 4.2 Document the command in the host CLI README next to `workspace open`, saying plainly which one a script wants; verify the README builds and the two commands are distinguishable from the text alone

## 5. Proof on this machine

- [x] 5.1 Run `swm workspace ensure` against a story with worktrees but no live workspace — `laio-plugin` is one today — and verify a socket appears, `swm pane list` reports its groups, and the shell that ran it is still the shell that ran it
- [x] 5.2 Verify `swm pane open -w "$(swm workspace ensure <story>)" -g <group>` opens a pane, which is the composition steward depends on
