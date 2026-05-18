## 1. swm/session-tmux — Fix broken spec scenario and test fixture

- [x] 1.1 In `plugins/session-tmux/internal/session/tmux_test.go` (`TestOpenPaneGroup_WithPaneGroupCommand`): update the `fakeHostClient` TOML fixture from `--config` to `--file` and add `--socket {{tmux_socket}}` to the command; update the `require.Contains` assertion to match the new expected string.
- [x] 1.2 In `openspec/specs/session-tmux/spec.md` (Scenario: Custom pane_group_command): update the WHEN/THEN text to use `--file` and `--socket {{tmux_socket}}` so the spec matches the actual laio CLI.

## 2. swm/session-tmux — Add `{{tmux_socket}}` template variable

- [x] 2.1 In `plugins/session-tmux/internal/session/tmux.go` (`paneGroupCommand`): add `{{tmux_socket}}` → `req.GetWorkspaceId()` to the substitution map alongside the existing `{{worktree_path}}`, `{{story_name}}`, and `{{project_id}}` entries.
- [x] 2.2 In `plugins/session-tmux/internal/session/tmux_test.go`: add `TestOpenPaneGroup_WithPaneGroupCommand_SocketSubstitution` that sets a TOML with `{{tmux_socket}}` in `pane_group_command` and asserts the substituted socket path appears in the logged tmux invocation.
- [x] 2.3 Run `task fmt && task lint && task test` in `plugins/session-tmux/` and confirm zero failures.

## 3. laio — Socket interface: `--socket` flag and `LAIO_TMUX_SOCKET` env var

- [x] 3.1 In `src/app/cli/command_line.rs` (`Commands::Start`): add `#[clap(long)] socket: Option<String>` field. In `Cli::run()`, resolve the socket value as `socket.or_else(|| std::env::var("LAIO_TMUX_SOCKET").ok())` and thread it into the muxer/session pipeline.
- [x] 3.2 Expose the resolved socket value through `SessionManager::start()` so it reaches the `Multiplexer::start()` call (add a parameter or context struct as needed to avoid global state).
- [x] 3.3 Add a unit test in `src/app/cli/` that verifies `--socket /tmp/test.sock` is forwarded to the multiplexer.
- [x] 3.4 Add a unit test verifying `LAIO_TMUX_SOCKET=/tmp/test.sock` is honoured when `--socket` is absent.

## 4. laio — Thread socket path through tmux client

- [x] 4.1 In the tmux client type (wherever `Command::new("tmux")` is built, e.g. `src/muxer/tmux/`): accept an `Option<String>` socket field. When `Some(path)`, prepend `-S <path>` to every tmux command that is constructed.
- [x] 4.2 Update `create_session`, `has_session`, `new_window`, `split_window`, `send_keys`, `setenv`, `attach_session`, `switch_client`, and any other tmux command builders to pass through the socket flag.
- [x] 4.3 Update existing tmux muxer tests to assert that `-S <path>` appears in command strings when a socket is configured.

## 5. laio — In-session mode (configure current session, no new-session)

- [x] 5.1 In `src/muxer/tmux/mux.rs` (`start()`): add an early branch — if `socket` is `Some` and `client.is_inside_session()` is `true`, call `configure_current_session()` instead of the normal `create_session` path and return.
- [x] 5.2 Implement `configure_current_session(&self, session: &Session, env_vars: &[(&str, &str)], skip_cmds: bool)`: sets env vars on the existing session, creates ALL configured windows via `new-window` (`force_new_windows = true`) so they outlive the laio process, flushes commands, then kills the original window that laio was launched in so focus lands on the correct configured window.
- [x] 5.3 Add a unit test `mux_start_in_session_mode`: set `is_inside_session()` to return `true`, provide a socket, call `start()`, and assert `new-session` is NOT called but `new-window` IS called for each configured window.
- [x] 5.4 Run `cargo test` in `laio-cli/` and confirm zero failures.

## 7. swm/session-tmux — Bootstrap session in OpenWorkspace

- [x] 7.1 In `plugins/session-tmux/internal/session/tmux.go` (`OpenWorkspace`): replace per-project session pre-creation with a single bootstrap session named after the story. This keeps the tmux server alive (tmux exits with `exit-empty on` when there are no sessions) without racing with `pane_group_command` — which requires the session to NOT exist yet so it can be the initial process.
- [x] 7.2 Update/add tests: `TestOpenWorkspace_SetsSWMStory` asserts `new-session` carries `-e SWM_STORY=<story>`; `TestOpenWorkspace_EmptyWorktreePaths` asserts the bootstrap session uses the story name with no `-c`; `TestOpenWorkspace_NoProjectSessionsPreCreated` asserts `github•com` sessions are NOT created during `OpenWorkspace`.
- [x] 7.3 Run `task fmt && task lint && task test` in `plugins/session-tmux/` and confirm zero failures.

## 6. Sync and documentation

- [x] 6.1 Run `openspec sync-specs --change laio-plugin-support` in the swm repo to promote the delta spec into `openspec/specs/session-tmux/spec.md`.
- [x] 6.2 Verify end-to-end manually: verified with global config (`~/.config/swm/laio.yaml`) using `--var path={{worktree_path}}` and `path: "{{ path }}"` in the YAML; sessions appear on the story's socket and windows open with the correct worktree CWD.
