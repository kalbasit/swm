## 1. Make the bug expressible in tests (test double)

- [x] 1.1 In `plugins/session-tmux/internal/session/testdata/faketmux/main.go`, make `new-session` record the value of its `-s` flag into a sessions file alongside the socket path (e.g. `<socket>.sessions`, one name per line), creating the file if absent.
- [x] 1.2 Add a `resolveTarget(target string, sessions []string) bool` helper to faketmux that mirrors tmux(1) target resolution: if `target` begins with `=`, strip it and match exactly only; otherwise try exact, then prefix, then `fnmatch`/`path.Match`. Comment that it deliberately mirrors the tmux(1) rules and why.
- [x] 1.3 Rewire faketmux's `has-session` case to read the sessions file and answer via `resolveTarget`, exiting 0 on match and 1 otherwise. Keep `FAKETMUX_HAS_SESSION` as an explicit short-circuit override so existing tests that set it are unaffected.
- [x] 1.4 Run the existing `plugins/session-tmux` suite and confirm it is still green — this step must not change any assertion outcomes.

## 2. Red — regression tests that fail against current code

- [x] 2.1 Add a test to `plugins/session-tmux/internal/session/tmux_test.go` covering spec scenario "Project whose name is a prefix of another project's name": call `OpenPaneGroup` for `<host>/name-two`, then for `<host>/name` on the same socket; assert two distinct `new-session` invocations were recorded and the two returned `PaneGroup.PaneGroupId` values differ.
- [x] 2.2 Add the reverse-order case (spec scenario "Creation order does not affect resolution"): create `<host>/name` first, then `<host>/name-two`, asserting the same.
- [x] 2.3 Add a test for spec scenario "Switching to a project whose name is a prefix of another": assert `SwitchTo` targets the requested pane group exactly.
- [x] 2.4 Assert the idempotent path still holds (spec scenario "Existing pane group is still reused"): a second `OpenPaneGroup` for an already-open project issues no new `new-session`.
- [x] 2.5 Run the suite and confirm 2.1–2.3 fail for the right reason (missing `new-session` / wrong target), not from harness error.

## 3. Green — exact-match targeting in session/tmux.go

- [x] 3.1 Add a small helper (e.g. `exactTarget(name string) string` returning `"=" + name`) in `plugins/session-tmux/internal/session/tmux.go` with a comment explaining tmux's exact→prefix→fnmatch resolution and why the escape is required.
- [x] 3.2 Apply it to `has-session -t` in `OpenPaneGroup` (~line 226). Leave `new-session -s <name>` unchanged — `-s` names the session being created, it is not a target.
- [x] 3.3 Apply it to `switch-client -t` (~line 304) and to the `attach-session -t` entry in `ExecArgv` (~line 311) in `SwitchTo`.
- [x] 3.4 Leave `kill-pane -t <paneID>` (~line 358) unescaped — pane IDs are tmux-assigned and unambiguous. Add a brief comment noting this is deliberate.
- [x] 3.5 Update the two existing argv assertions in `tmux_test.go` (~lines 232 and 695) that expect an unescaped `attach-session -t <name>` to expect `=<name>`.
- [x] 3.6 Run the suite; tasks 2.1–2.4 must now pass.

## 4. Green — exact-match targeting in layout/apply.go

- [x] 4.1 Apply exact-match targeting to the name-based targets in `plugins/session-tmux/internal/layout/apply.go`: `setenv -t <sessionName>` (~line 28), `rename-window -t <sessionName>:0` (~line 52, becoming `=<sessionName>:0`), and `new-window -t <sessionName>` (~line 57).
- [x] 4.2 In `paneID`/`display-message -t <target>` (~line 209), escape the target only on the session-name path; leave it as-is when the target is already a pane ID.
- [x] 4.3 Leave the pane-ID targets unescaped: `select-pane` (~86), `resize-pane` (~94), `send-keys` (~201), and `targetArgs` (~227).
- [x] 4.4 Update any `layout` package test assertions that match on the affected argv, and run the `layout` suite.

## 5. Verify

- [x] 5.1 Run `task fmt` and confirm exit status 0, applying and re-running if it produces changes.
- [x] 5.2 Run `task lint` and confirm exit status 0.
- [x] 5.3 Run `task test` and confirm exit status 0 across all modules.
- [x] 5.4 Cross-check each scenario in `specs/session-tmux/spec.md` against a corresponding test, confirming all five ADDED scenarios and the two changed `SwitchTo` scenarios are covered.
- [x] 5.5 Confirm no proto files under `proto/` were touched, so no nix vendorHash update is needed.
