## 1. Failing Tests (Red)

- [x] 1.1 `plugins/session-tmux`: Add integration test asserting that after `OpenWorkspace`, a command run inside the tmux session does not see `SWM_HOST_SOCKET`, `SWM_LOG_LEVEL`, or `SWM_PLUGIN_MAGIC_COOKIE` in its environment
- [x] 1.2 `plugins/session-tmux`: Add integration test asserting that pre-existing user environment variables (e.g. `HOME`, `PATH`, a custom sentinel var) are present and unchanged inside the tmux session after `OpenWorkspace`

## 2. Implementation (Green)

- [x] 2.1 `plugins/session-tmux`: Extract a `filteredEnv()` helper that takes `os.Environ()` and removes `SWM_HOST_SOCKET`, `SWM_LOG_LEVEL`, and `SWM_PLUGIN_MAGIC_COOKIE`, returning the cleaned slice
- [x] 2.2 `plugins/session-tmux`: Set `cmd.Env = filteredEnv()` on every `exec.Cmd` that launches a tmux server process (the `run()` helper and any other direct tmux invocations at workspace creation)

## 3. Verification

- [x] 3.1 Run `task test` — confirm the two new integration test scenarios pass and no existing tests regress
- [x] 3.2 Run `task fmt` and `task lint` — confirm zero issues
