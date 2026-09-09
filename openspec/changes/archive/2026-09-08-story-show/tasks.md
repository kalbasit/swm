## 1. The command

- [x] 1.1 Add `swm story show [<story-name>] [--json]` resolving the story from argument, `$SWM_STORY`, then `default_story`; verify each precedence step with a test
- [x] 1.2 Report the branch name, creation time, and every attached project with the path `layout.Resolver.WorktreePath` gives it; verify the reported path matches the resolver rather than a path built in the command
- [x] 1.3 Verify the default story reports canonical clones rather than worktree paths, which is the resolver's existing special case and must not be reimplemented here
- [x] 1.4 Refuse an unknown story non-zero, naming it; verify no record is printed
- [x] 1.5 Report a story with no attached projects as a story with an empty project list; verify it is not an error
- [x] 1.6 Verify no session plugin is loaded, so the command answers with no multiplexer running

## 2. Output

- [x] 2.1 `--json` prints one object with the story and its projects; verify it parses and carries a project key and worktree path per project
- [x] 2.2 Without `--json`, print a readable form; verify the branch name and every project path appear

## 3. Surface

- [x] 3.1 Register story-name completion for the positional argument; verify Tab offers story names, not filenames, and that a store error offers nothing
- [x] 3.2 Document the command in the host CLI README next to `story list`; verify the text says plainly that paths come from swm's records rather than the filesystem

## 4. Proof on this machine

- [x] 4.1 Run `swm story show steward --json` against this machine's real store and verify the branch name matches `swm config get story.branch_name_template` rendered for that story, and every path exists
- [x] 4.2 Verify a story with no live workspace still reports in full, since that is the case steward hits before it opens one
