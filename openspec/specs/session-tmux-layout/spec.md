# session-tmux-layout Specification

## Purpose
Specifies the built-in tmux window/pane layout system for `session-tmux`. Covers two-tier config resolution (per-repo `.swm/session-tmux.toml` > global `$XDG_CONFIG_HOME/swm/session-tmux.toml` > built-in default), the TOML schema, `text/template` variable substitution, the recursive flex-split algorithm, and per-pane command/focus/zoom application.
## Requirements
### Requirement: Layout config resolution
`session-tmux` SHALL resolve a layout config for `OpenPaneGroup` using the following priority order (first match wins):

1. `pane_group_command` set in `config.toml` — existing behavior; a warning SHALL be logged if a layout config file exists at either tier and is being ignored.
2. `<worktree_path>/.swm/session-tmux.toml` — per-repo config (overrides global).
3. `$XDG_CONFIG_HOME/swm/session-tmux.toml` — global config (user-wide default).
4. Built-in default — `$EDITOR` (or `vim` if unset) in the first window, a shell in the second.

Only the first matching source is used. Configs are never merged across tiers.

#### Scenario: Per-repo config used when present
- **WHEN** `<worktree_path>/.swm/session-tmux.toml` exists and `pane_group_command` is not set
- **THEN** the per-repo config is loaded and applied; the global config (if any) is not read

#### Scenario: Global config used when no per-repo config
- **WHEN** `$XDG_CONFIG_HOME/swm/session-tmux.toml` exists, `pane_group_command` is not set, and no per-repo config exists
- **THEN** the global config is loaded and applied

#### Scenario: Per-repo config wins over global
- **WHEN** both `<worktree_path>/.swm/session-tmux.toml` and `$XDG_CONFIG_HOME/swm/session-tmux.toml` exist and `pane_group_command` is not set
- **THEN** only the per-repo config is applied; the global config is not read

#### Scenario: pane_group_command wins and warning logged
- **WHEN** `pane_group_command` is set in `config.toml` and `<worktree_path>/.swm/session-tmux.toml` also exists
- **THEN** `pane_group_command` is used and a warning is logged naming the ignored layout config file

#### Scenario: No config at either tier falls back to default
- **WHEN** neither `pane_group_command` nor any layout config file is present
- **THEN** the default layout is applied (`$EDITOR` in window 1, shell in window 2)

---

### Requirement: Layout config TOML schema
The layout config file SHALL be valid TOML conforming to this schema:

```toml
# Top-level (session-scoped) fields — all optional
path         = string          # base path; defaults to worktree_path
shell        = string          # shell binary for panes without an explicit command
pane_cmd_delay = int           # milliseconds to wait between send-keys calls (default: 0)

[env]                          # environment variables injected into every pane
KEY = "value"

[[startup]]                    # commands run in the first pane of the first window before layout
command = string
args    = [string]

[[windows]]                    # at least one window required
name           = string        # required; must be non-empty
path           = string        # optional; resolved relative to session path
flex_direction = "row"|"column" # direction for child pane splits (default: "column")

  [[windows.panes]]            # optional; defaults to one full-size pane
  flex           = int         # relative size weight (default: 1, minimum: 1)
  flex_direction = "row"|"column"
  path           = string
  zoom           = bool        # zoom this pane at creation time (default: false)
  focus          = bool        # focus this pane at creation time (default: false)
  shell          = string
  commands       = [string]    # commands sent via send-keys

  [windows.panes.env]
  KEY = "value"

    [[windows.panes.panes]]    # nested panes (recursive)
    # same fields as [[windows.panes]]
```

Constraints:
- At least one `[[windows]]` entry is required.
- `flex` must be ≥ 1.
- At most one pane per window tree may have `focus = true`.
- At most one pane per window tree may have `zoom = true`.

#### Scenario: Config with no windows is rejected
- **WHEN** a layout config file contains no `[[windows]]` entries
- **THEN** `OpenPaneGroup` returns an `InvalidArgument` error before any tmux commands are issued

#### Scenario: Config with flex less than 1 is rejected
- **WHEN** a layout config file contains a pane with `flex = 0`
- **THEN** `OpenPaneGroup` returns an `InvalidArgument` error before any tmux commands are issued

#### Scenario: Config with two focused panes in the same window is rejected
- **WHEN** a window's pane tree contains two panes both with `focus = true`
- **THEN** `OpenPaneGroup` returns an `InvalidArgument` error before any tmux commands are issued

---

### Requirement: Template variable substitution in layout config
Before parsing, the raw TOML bytes SHALL be rendered through `text/template` with the following variables auto-injected:

| Variable           | Value                                           |
|--------------------|-------------------------------------------------|
| `{{.WorktreePath}}`| The `worktree_path` from `OpenPaneGroupRequest` |
| `{{.StoryName}}`   | The story name from `OpenPaneGroupRequest`      |
| `{{.TmuxSocket}}`  | The workspace socket path (`workspace_id`)      |

#### Scenario: WorktreePath substituted in window path
- **WHEN** a layout config contains `path = "{{.WorktreePath}}/src"` and `worktree_path` is `/home/user/code/stories/feat/github.com/org/repo`
- **THEN** the resolved window path is `/home/user/code/stories/feat/github.com/org/repo/src`

#### Scenario: StoryName substituted in a command
- **WHEN** a layout config contains `commands = ["echo {{.StoryName}}"]` and `story_name` is `feat-x`
- **THEN** `send-keys` is called with `echo feat-x`

#### Scenario: TmuxSocket substituted
- **WHEN** a layout config contains `commands = ["tmux -S {{.TmuxSocket}} list-sessions"]` and `workspace_id` is `/run/user/1000/swm/tmux/feat-x.sock`
- **THEN** `send-keys` is called with `tmux -S /run/user/1000/swm/tmux/feat-x.sock list-sessions`

---

### Requirement: Layout application — windows
`session-tmux` SHALL apply the layout config to a newly created tmux session by:
1. Renaming the default window to the first window's `name`.
2. Setting the first window's working directory to the resolved `path`.
3. Creating subsequent windows with `new-window -n <name> -c <path>`.

Window creation order matches the order of `[[windows]]` entries in the config.

#### Scenario: Single window created with correct name and path
- **WHEN** a layout config has one window with `name = "editor"` and `path = "{{.worktree_path}}"` and `OpenPaneGroup` creates a new session
- **THEN** the tmux session has exactly one window named `editor` with working directory set to `worktree_path`

#### Scenario: Multiple windows created in order
- **WHEN** a layout config has windows named `["editor", "shell", "test"]` and `OpenPaneGroup` creates a new session
- **THEN** the tmux session has three windows in that order with those names

---

### Requirement: Layout application — pane splits
`session-tmux` SHALL split panes using a recursive percentage algorithm:

For each group of sibling panes (ordered by position in config):
- `totalFlex = Σ flex(sibling)`
- For each sibling except the last:
  - `remainingFlex = Σ flex(following siblings)`
  - `splitPercent = clamp(remainingFlex * 100 / (flex(current) + remainingFlex), 1, 99)`
  - Run `tmux split-window -p <splitPercent>` (add `-h` for `flex_direction = "row"`)
- The last sibling takes whatever space remains.

Rounding error accumulates to the last pane.

#### Scenario: Two equal panes split 50/50
- **WHEN** a window has two panes each with `flex = 1` and `flex_direction = "column"` (default)
- **THEN** `split-window -p 50` is called once, producing two vertically equal panes

#### Scenario: Three equal panes split into thirds
- **WHEN** a window has three panes each with `flex = 1`
- **THEN** `split-window -p 66` is called first (leaving ~34% for the first pane via floor division), then `split-window -p 50` on the remainder

#### Scenario: Weighted flex produces proportional splits
- **WHEN** a window has two panes with `flex = 2` and `flex = 1`
- **THEN** `split-window -p 33` is called, giving the first pane ~67% and the second ~33%

#### Scenario: Row direction uses horizontal split
- **WHEN** a window has panes with `flex_direction = "row"`
- **THEN** `split-window -h` is used instead of the default vertical split

#### Scenario: Extreme flex ratio is clamped
- **WHEN** a pane split would compute a `splitPercent` of 0 or 100
- **THEN** the value is clamped to 1 or 99 respectively

---

### Requirement: Layout application — pane commands, focus, and zoom
After creating each pane, `session-tmux` SHALL:
- Send each entry in `commands` via `tmux send-keys <cmd> Enter`, waiting `pane_cmd_delay` ms between each.
- If `focus = true`: run `tmux select-pane -t <pane>` after all panes in the window are created.
- If `zoom = true`: run `tmux resize-pane -Z -t <pane>` after focus is applied.
- Both `focus` and `zoom` are applied at creation time only; `SwitchTo` does not re-apply them.

#### Scenario: Commands sent to pane
- **WHEN** a pane has `commands = ["git status", "git log --oneline"]`
- **THEN** `send-keys "git status" Enter` and `send-keys "git log --oneline" Enter` are called in sequence for that pane

#### Scenario: pane_cmd_delay respected between commands
- **WHEN** `pane_cmd_delay = 200` and a pane has two commands
- **THEN** a 200 ms delay is observed between the two `send-keys` calls

#### Scenario: focus pane selected after window layout
- **WHEN** a window's third pane has `focus = true`
- **THEN** `select-pane` targets that pane after all splits and commands in the window are applied

#### Scenario: zoom applied after focus
- **WHEN** a pane has both `focus = true` and `zoom = true`
- **THEN** `select-pane` is called first, then `resize-pane -Z`

#### Scenario: SwitchTo does not re-apply focus or zoom
- **WHEN** `SwitchTo` is called for a session whose layout config has `focus = true` on a pane
- **THEN** no `select-pane` command is issued by `SwitchTo`

