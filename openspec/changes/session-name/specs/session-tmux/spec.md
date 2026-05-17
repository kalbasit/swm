## MODIFIED Requirements

### Requirement: Socket-per-workspace model
`session-tmux` SHALL map each swm workspace to a dedicated tmux server socket at `$XDG_RUNTIME_DIR/swm/tmux/<story-name>.sock`. Each pane group within a workspace SHALL map to a tmux session (named by sanitizing the full canonical path `host/seg1/.../segN` to be tmux-safe — replacing `.` with `•` (U+2022) and `:` with `：` (U+FF1A), e.g., `github•com/kalbasit/swm` for `github.com/kalbasit/swm`) within that socket. This preserves the v1 tmux isolation model while preventing collisions between same-named repos from different forges or orgs.

#### Scenario: Workspace socket path
- **WHEN** `OpenWorkspace({story_name: "feat-x", ...})` is called
- **THEN** the tmux server is started (if not running) on socket `$XDG_RUNTIME_DIR/swm/tmux/feat-x.sock`

#### Scenario: Pane group session name
- **WHEN** `OpenPaneGroup({story_name: "feat-x", project_id: {host: "github.com", segments: ["kalbasit", "swm"]}, ...})` is called
- **THEN** a tmux session named `github•com/kalbasit/swm` is created within the `feat-x.sock` server

#### Scenario: Session name collision prevention
- **WHEN** `OpenPaneGroup` is called for two projects with the same repo name but different orgs — `{host: "github.com", segments: ["org-a", "utils"]}` and `{host: "github.com", segments: ["org-b", "utils"]}` — within the same workspace
- **THEN** two distinct sessions `github•com/org-a/utils` and `github•com/org-b/utils` are created
