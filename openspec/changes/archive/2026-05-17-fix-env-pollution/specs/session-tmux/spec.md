## ADDED Requirements

### Requirement: Environment isolation at workspace launch

Before launching the tmux server process, the session plugin MUST explicitly construct the child process environment using a denylist approach: start from the inherited `os.Environ()` and strip all plugin-internal variables. The resulting environment MUST NOT contain `SWM_HOST_SOCKET`, `SWM_LOG_LEVEL`, or `SWM_PLUGIN_MAGIC_COOKIE`.

The canonical environment variable ownership model for a user session:

| Variable | Present in tmux session |
|---|:---:|
| `SWM_HOST_SOCKET` | no — stripped |
| `SWM_LOG_LEVEL` | no — stripped |
| `SWM_PLUGIN_MAGIC_COOKIE` | no — stripped |
| `SWM_STORY` | yes — set by session plugin at workspace open |
| All other user env vars | yes — inherited unchanged |

#### Scenario: Plugin-internal vars absent from new tmux window
- **WHEN** a workspace is opened via `OpenWorkspace` and a new shell is spawned in a tmux window
- **THEN** `SWM_HOST_SOCKET` is absent from the shell's environment
- **AND** `SWM_LOG_LEVEL` is absent from the shell's environment
- **AND** `SWM_PLUGIN_MAGIC_COOKIE` is absent from the shell's environment

#### Scenario: User environment preserved in tmux session
- **WHEN** a workspace is opened and the user had `HOME`, `PATH`, and arbitrary user-defined vars set before invoking swm
- **THEN** those variables are present and unchanged in the tmux session's shell environment

#### Scenario: SWM_STORY present in tmux session
- **WHEN** a workspace is opened for story `<story-name>`
- **THEN** `SWM_STORY` is set to `<story-name>` in the tmux session environment
