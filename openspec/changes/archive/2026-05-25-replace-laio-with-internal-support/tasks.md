## 1. Layout config types and parsing (plugins/session-tmux)

- [x] 1.1 Create `plugins/session-tmux/internal/layout/config.go` — define `Config`, `Window`, `Pane`, `Command` TOML structs with all fields from the spec schema
- [x] 1.2 Add `LoadConfig(worktreePath, xdgConfigHome string) (*Config, error)` that implements the two-tier resolution (per-repo → global → nil)
- [x] 1.3 Implement `text/template` rendering of raw TOML bytes before parsing, injecting `worktree_path`, `story_name`, `tmux_socket`
- [x] 1.4 Add config validation: at least one window, `flex >= 1`, at most one `focus` and one `zoom` per window tree; return `InvalidArgument` on failure
- [x] 1.5 Write table-driven unit tests for `LoadConfig` covering: per-repo wins, global fallback, both absent, template substitution, all validation error cases

## 2. Flex-split algorithm (plugins/session-tmux)

- [x] 2.1 Create `plugins/session-tmux/internal/layout/split.go` — implement `splitPercent(currentFlex, remainingFlex int) int` with [1, 99] clamping
- [x] 2.2 Write unit tests for `splitPercent`: equal pairs, thirds, weighted ratios, extreme values that trigger clamping

## 3. Layout application engine (plugins/session-tmux)

- [x] 3.1 Create `plugins/session-tmux/internal/layout/apply.go` — implement `Apply(ctx context.Context, run RunFunc, sock, sessionName string, cfg *Config) error` that orchestrates all tmux commands
- [x] 3.2 Implement window creation: rename default window, create additional windows with `new-window -n <name> -c <path>`
- [x] 3.3 Implement recursive pane splitting: depth-first traversal, `split-window -p <pct> [-h]` per the flex algorithm
- [x] 3.4 Implement post-split actions per pane: `send-keys` for each command (with `pane_cmd_delay` sleep), `select-pane` for `focus`, `resize-pane -Z` for `zoom`
- [x] 3.5 Implement `env` injection: set variables via `tmux setenv -t <session>` before window creation
- [x] 3.6 Implement `startup` commands: send to first pane of first window before other layout steps
- [x] 3.7 Write unit tests for `Apply` using a fake `RunFunc` that records commands; cover: single window/pane, multi-window, nested panes, focus, zoom, commands with delay, row vs column direction

## 4. Wire layout into OpenPaneGroup (plugins/session-tmux)

- [x] 4.1 In `plugins/session-tmux/internal/session/tmux.go`, update `OpenPaneGroup` to call `layout.LoadConfig` after session creation; apply resolved config or fall back to default layout
- [x] 4.2 Add warning log when `pane_group_command` is set and a layout config file also exists at either tier
- [x] 4.3 Update existing `OpenPaneGroup` unit tests in `tmux_test.go` to cover: per-repo layout applied, global layout fallback, per-repo wins over global, `pane_group_command` wins with warning, no config → default layout unchanged

## 5. Update specs (openspec)

- [ ] 5.1 Run `openspec sync --change replace-laio-with-internal-support` to merge delta specs into main specs
- [ ] 5.2 Verify `openspec/specs/session-tmux/spec.md` reflects the updated `OpenPaneGroup` requirement
- [ ] 5.3 Verify `openspec/specs/session-tmux-layout/spec.md` is created as a new main spec

## 6. Verification

- [ ] 6.1 Run `task fmt` and confirm zero diff
- [ ] 6.2 Run `task lint` and confirm zero findings
- [ ] 6.3 Run `task test` and confirm all tests pass
