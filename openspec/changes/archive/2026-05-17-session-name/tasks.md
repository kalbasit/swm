## 1. Fix session name derivation (plugins/session-tmux)

- [x] 1.1 In `plugins/session-tmux/internal/session/tmux.go`, rewrite `sessionName(key string)` to sanitize the full key (`.` → `•`, `:` → `：`) instead of returning just the last segment
- [x] 1.2 In `OpenPaneGroup` (`tmux.go`), replace `name := segments[len(segments)-1]` with a call to `sessionName(pid.GetHost() + "/" + strings.Join(pid.GetSegments(), "/"))` and remove the now-redundant `len(segments) == 0` guard (replace with a check that host is also non-empty)

## 2. Update tests (plugins/session-tmux)

- [x] 2.1 In `plugins/session-tmux/internal/session/tmux_test.go`, update all assertions that reference bare basename session names (e.g. `"swm"`) to use the full sanitized path (e.g. `"github•com/kalbasit/swm"`)
- [x] 2.2 Add a table-driven test case to the `sessionName` function covering: dots replaced, colons replaced, slashes preserved, and a collision-prevention case (two different orgs, same repo name)

## 3. Verify

- [x] 3.1 Run `task fmt` in `plugins/session-tmux` and confirm zero diff
- [x] 3.2 Run `task lint` in `plugins/session-tmux` and confirm zero findings
- [x] 3.3 Run `task test` in `plugins/session-tmux` and confirm all tests pass
