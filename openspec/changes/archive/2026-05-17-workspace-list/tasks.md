## 1. Command skeleton

- [x] 1.1 [cmd/swm] Create `cmd/swm/internal/cmd/workspace_list.go` with a `newWorkspaceListCmd` cobra command factory
- [x] 1.2 [cmd/swm] Wire `workspace list` into the existing `workspace` cobra command group (alongside `workspace open`)
- [x] 1.3 [cmd/swm] Inject the story store into the command using the same pattern as `story list`

## 2. Core implementation

- [x] 2.1 [cmd/swm] Read all stories from the story store; propagate any error to stderr and exit non-zero
- [x] 2.2 [cmd/swm] Sort story names lexicographically; sort project canonical paths (`host/segments...`) lexicographically within each story
- [x] 2.3 [cmd/swm] Implement `renderWorkspaceTree(w io.Writer, stories []...)` that writes the glyph tree to any writer:
  ```
  <workspace>
  ├── <project>
  └── <project>
  ```
  Omit the connector line and child entries for workspaces with no projects.

## 3. Tests

- [x] 3.1 [cmd/swm] Unit-test `renderWorkspaceTree` with a table-driven test covering: empty store, workspace with no projects, single workspace + one project, multiple workspaces + multiple projects
- [x] 3.2 [cmd/swm] Integration-test `swm workspace list` against a temp story store: seed known stories and projects, run the command, assert exact stdout output and zero exit code
- [x] 3.3 [cmd/swm] Integration-test store error path: stub a failing store, assert non-zero exit and error on stderr
