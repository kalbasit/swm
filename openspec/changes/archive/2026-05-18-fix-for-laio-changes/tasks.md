## 1. Fix test constants (plugins/session-tmux)

- [x] 1.1 In `plugins/session-tmux/internal/session/tmux_test.go`, update `testLaioPaneGroupCommandTOML` to insert `--replace-current-session` between `--tmux-socket {{tmux_socket}}` and `--skip-attach`
- [x] 1.2 In the same file, update the `wantCmd` string literal in `TestOpenPaneGroup_WithPaneGroupCommand_BasicCommandSubstitution` (around line 379) to include `--replace-current-session` before `--skip-attach`
- [x] 1.3 Run `task test` in `plugins/session-tmux/` and confirm all tests pass

## 2. Update documentation (plugins/session-tmux)

- [x] 2.1 In `plugins/session-tmux/README.md`, update the per-repo example `pane_group_command` (line ~99) to add `--replace-current-session` before `--skip-attach`
- [x] 2.2 In `plugins/session-tmux/README.md`, update the shared-layout example `pane_group_command` (line ~125) to add `--replace-current-session` before `--skip-attach`
- [x] 2.3 Update the inline explanation below the examples (line ~132–134) to note that `--replace-current-session` is required for laio to patch the existing session in place rather than create a new one

## 3. Update live user config (~/.config/swm)

- [x] 3.1 In `~/.config/swm/config.toml`, add a `[plugins.config.session-tmux]` section with `pane_group_command = "laio start --file ~/.config/swm/laio.yaml --tmux-socket '{{tmux_socket}}' --replace-current-session --skip-attach --var path='{{worktree_path}}'"`

## 4. Verify

- [x] 4.1 Run `task fmt` from the repo root and confirm exit 0
- [x] 4.2 Run `task lint` from the repo root and confirm exit 0
- [x] 4.3 Run `task test` from the repo root and confirm exit 0
