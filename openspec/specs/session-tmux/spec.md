## Purpose

The `session-tmux` plugin manages per-story tmux servers for swm. Each workspace (story) gets a dedicated tmux socket, and each pane group (project worktree) gets a named session within that socket. The plugin handles workspace lifecycle (create, attach, close), session navigation (SwitchTo), context detection (IsInsideWorkspace, CurrentContext), and optional per-session layout via `pane_group_command`.

## Requirements

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

### Requirement: OpenWorkspace creates and attaches
`session-tmux` SHALL implement `Session.OpenWorkspace({story_name, worktree_paths})` by starting the tmux server socket if it does not exist. When the server is started, a single bootstrap session named after the story SHALL be created to keep the server alive (tmux's `exit-empty on` default exits the server when there are no sessions). Project sessions SHALL NOT be pre-created; they are created lazily by `OpenPaneGroup` so that `pane_group_command` is applied to each one individually. If the socket already exists, the call is idempotent.

#### Scenario: New workspace
- **WHEN** `OpenWorkspace` is called for a story with no existing socket
- **THEN** a new tmux server is started on the story's socket, a single bootstrap session named after the story is created, and `Workspace` is returned

#### Scenario: Existing workspace
- **WHEN** `OpenWorkspace` is called and the story's socket already has a running server
- **THEN** the call completes without creating duplicate sessions

#### Scenario: Returns Workspace proto
- **WHEN** `OpenWorkspace` completes successfully
- **THEN** a `Workspace` message with `name = story_name` and `id = socket_path` is returned

### Requirement: CloseWorkspace terminates server
`session-tmux` SHALL implement `Session.CloseWorkspace({story_name})` by sending `tmux -S <socket> kill-server`. The socket file SHALL be cleaned up.

#### Scenario: Close running workspace
- **WHEN** `CloseWorkspace({story_name: "feat-x"})` is called and the socket is active
- **THEN** `tmux kill-server` is run on the socket and the socket file is removed

#### Scenario: Close non-existent workspace
- **WHEN** `CloseWorkspace` is called for a story with no socket file
- **THEN** the call succeeds (idempotent) with no error

### Requirement: ListWorkspaces streams active sockets
`session-tmux` SHALL implement `Session.ListWorkspaces()` by scanning `$XDG_RUNTIME_DIR/swm/tmux/` for socket files, probing each with `tmux -S <socket> list-sessions -F ""` to confirm the server is alive, and streaming one `Workspace` message per live socket.

#### Scenario: Multiple active workspaces
- **WHEN** `ListWorkspaces()` is called and two story sockets are live
- **THEN** two `Workspace` messages are streamed

#### Scenario: Stale socket files ignored
- **WHEN** a socket file exists but the tmux server is no longer running
- **THEN** that socket is excluded from the streamed results

### Requirement: paneGroupCommand exposes template variables
`session-tmux` SHALL render `pane_group_command` through Go `text/template` with `{{.WorktreePath}}`, `{{.StoryName}}`, `{{.ProjectID}}`, and `{{.TmuxSocket}}` before executing it. `{{.TmuxSocket}}` expands to the absolute path of the story's tmux socket (the same value as `workspace_id` in the request).

#### Scenario: Template variables are substituted
- **WHEN** `config.toml` has `pane_group_command = "my-layout --socket '{{.TmuxSocket}}' --path '{{.WorktreePath}}'"` and `OpenPaneGroup` is called with `workspace_id = /run/user/1000/swm/tmux/feat-x.sock` and `worktree_path = /home/user/code/stories/feat-x/github.com/org/repo`
- **THEN** the command runs as `my-layout --socket '/run/user/1000/swm/tmux/feat-x.sock' --path '/home/user/code/stories/feat-x/github.com/org/repo'`

#### Scenario: Template substitution absent when no pane_group_command configured
- **WHEN** no `pane_group_command` is set in `config.toml`
- **THEN** the default layout engine is used and no template substitution occurs

### Requirement: OpenPaneGroup in existing workspace
`session-tmux` SHALL implement `Session.OpenPaneGroup({story_name, project_id, worktree_path})` by creating a new tmux session for the project within the story's socket (if it doesn't exist). The initial working directory SHALL be `worktree_path`.

`OpenPaneGroup` SHALL resolve the layout using the following priority order (first match wins):

1. If `pane_group_command` is set in `config.toml`: run that command (existing behavior). A warning SHALL be logged if a layout config file also exists at either tier.
2. If `<worktree_path>/.swm/session-tmux.toml` exists: apply the per-repo layout (see `session-tmux-layout` spec).
3. If `$XDG_CONFIG_HOME/swm/session-tmux.toml` exists: apply the global layout (see `session-tmux-layout` spec).
4. Default: run `$EDITOR` (or `vim` if unset) in the first window and a shell in the second.

When `pane_group_command` is configured, `session-tmux` SHALL validate that the first token of the command resolves to an executable via PATH lookup before creating the tmux session. If the binary is not found, `OpenPaneGroup` SHALL return a `FailedPrecondition` error naming the missing binary — no tmux session SHALL be created.

#### Scenario: Default layout
- **WHEN** no `pane_group_command` is configured and no layout config exists at either tier
- **THEN** the tmux session is created with two windows: the first running `$EDITOR` (or `vim`), the second running a shell

#### Scenario: Custom pane_group_command
- **WHEN** `config.toml` has `pane_group_command = "my-layout --socket '{{.TmuxSocket}}' --path '{{.WorktreePath}}'"` and `OpenPaneGroup` is called
- **THEN** the session's first window runs `my-layout` with both `{{.TmuxSocket}}` and `{{.WorktreePath}}` expanded to their respective values

#### Scenario: Per-repo layout config applied
- **WHEN** `<worktree_path>/.swm/session-tmux.toml` exists and `pane_group_command` is not set
- **THEN** the layout defined in that file is applied to the newly created tmux session

#### Scenario: Global layout config applied
- **WHEN** `$XDG_CONFIG_HOME/swm/session-tmux.toml` exists, `pane_group_command` is not set, and no per-repo config exists
- **THEN** the layout defined in the global config is applied to the newly created tmux session

#### Scenario: Per-repo layout wins over global
- **WHEN** both `<worktree_path>/.swm/session-tmux.toml` and `$XDG_CONFIG_HOME/swm/session-tmux.toml` exist and `pane_group_command` is not set
- **THEN** only the per-repo config is applied

#### Scenario: pane_group_command wins when layout config also present
- **WHEN** `pane_group_command` is set and `<worktree_path>/.swm/session-tmux.toml` also exists
- **THEN** `pane_group_command` is used, a warning is logged naming the ignored layout file, and the layout config is not read

#### Scenario: Idempotent for existing session
- **WHEN** `OpenPaneGroup` is called for a project whose session already exists on the socket
- **THEN** the existing session is reused and no new session is created

#### Scenario: pane_group_command binary not found
- **WHEN** `pane_group_command` is set to a command whose binary does not exist in `PATH`
- **THEN** `OpenPaneGroup` returns a `FailedPrecondition` error naming the missing binary, and no tmux session is created

### Requirement: SwitchTo switches active pane group

`session-tmux` SHALL implement `Session.SwitchTo({workspace_id, pane_group_id, close_origin_workspace_id, close_origin_pane_id})` by running `tmux -S <target_socket> switch-client -t =<session-name>` if already inside a tmux session, or building an `exec_argv` of `["tmux", "-S", "<target_socket>", "attach-session", "-t", "=<session-name>"]` otherwise. The leading `=` requests exact-match target resolution (see "Exact-match tmux target resolution").

When `close_origin_pane_id` is non-empty, the plugin SHALL:
1. Look up the socket path for `close_origin_workspace_id` in its workspace registry.
2. If the workspace is not found, return an error.
3. After performing the switch (or building `exec_argv`), run `tmux -S <origin_socket> kill-pane -t <close_origin_pane_id>`.
4. Ignore any "no such pane" or "no such session" errors from `kill-pane`.

`close_origin_pane_id` is a tmux-assigned pane ID, not a name, and is passed through unescaped.

The kill MUST happen inside the RPC handler before the response is returned, so that it executes even when the host will subsequently call `syscall.Exec` with the returned `exec_argv`.

#### Scenario: Switch when inside tmux
- **WHEN** `SwitchTo` is called from within an active tmux session
- **THEN** `tmux switch-client` is used to jump to the target session, with the target requested as an exact match

#### Scenario: Attach when outside tmux
- **WHEN** `SwitchTo` is called from a terminal not inside any tmux session
- **THEN** `exec_argv` of `["tmux", "-S", "<socket>", "attach-session", "-t", "=<session>"]` is returned and the host execs it

#### Scenario: Kill origin pane after in-place switch
- **WHEN** `SwitchTo` is called from inside a tmux session with non-empty `close_origin_workspace_id` and `close_origin_pane_id`
- **THEN** after `tmux switch-client` completes, `tmux kill-pane -t <close_origin_pane_id>` is run on the origin socket, and `SwitchToResponse` is returned with empty `exec_argv`

#### Scenario: Kill origin pane on exec path
- **WHEN** `SwitchTo` is called from outside any tmux session with non-empty `close_origin_workspace_id` and `close_origin_pane_id`
- **THEN** `kill-pane` runs on the origin socket before the `exec_argv` response is returned

#### Scenario: Kill origin — pane already gone
- **WHEN** `SwitchTo` is called with `close_origin_pane_id` set but the pane no longer exists
- **THEN** the "no such pane" error from `tmux kill-pane` is ignored and `SwitchTo` returns success

#### Scenario: Kill origin — unknown workspace
- **WHEN** `SwitchTo` is called with a `close_origin_workspace_id` that is not present in the plugin's workspace registry
- **THEN** `SwitchTo` returns a `NotFound` gRPC error

#### Scenario: No kill when close_origin_pane_id is empty
- **WHEN** `SwitchTo` is called with empty `close_origin_pane_id`
- **THEN** no `kill-pane` command is run and behaviour is identical to the existing switch

### Requirement: IsInsideWorkspace detection
`session-tmux` SHALL implement `Session.IsInsideWorkspace()` by checking whether `$TMUX` is set and the socket path matches `$XDG_RUNTIME_DIR/swm/tmux/<story>.sock` for any known story. Returns `BoolValue{value: true}` if inside a swm-managed workspace.

#### Scenario: Inside swm tmux workspace
- **WHEN** `IsInsideWorkspace()` is called with `$TMUX` pointing to a swm workspace socket
- **THEN** `BoolValue{value: true}` is returned

#### Scenario: Outside any tmux
- **WHEN** `IsInsideWorkspace()` is called with `$TMUX` unset
- **THEN** `BoolValue{value: false}` is returned

### Requirement: CurrentContext returns active workspace and pane group
`session-tmux` SHALL implement `Session.CurrentContext()` by reading `$TMUX` for the socket path and `$TMUX_PANE`/`tmux display-message` for the active session name. Returns a `CurrentContextResponse` with `workspace_id` and `pane_group_id`.

#### Scenario: Inside a swm workspace
- **WHEN** `CurrentContext()` is called from within a swm-managed tmux session
- **THEN** `CurrentContextResponse` is returned with the story name derived from the socket path and the pane group name from the active session

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

### Requirement: Exact-match tmux target resolution

`session-tmux` SHALL address every existing tmux session and window by literal name, so that
a target name is never resolved to a different session or window whose name merely starts
with, matches a glob against, or contains the requested name.

tmux resolves an unescaped `-t` target by trying, in order: exact name, prefix match, then
fnmatch. Because pane-group names are derived from project IDs, a project whose name is
a prefix of another project's name on the same host (for example `git.example.com/name` and
`git.example.com/name-two`) is therefore ambiguous unless the target is escaped. Every command
in which `session-tmux` passes a *name* as a target SHALL request exact matching; this covers
at minimum session existence checks, client switching, session attachment, environment
setting, window renaming, window creation, and pane-ID resolution.

Targets that are tmux-assigned identifiers rather than names — pane IDs of the form `%N` — are
already unambiguous and are exempt.

This requirement constrains only how existing objects are *addressed*. The names under which
sessions and windows are *created* are unchanged, so pane groups created by earlier versions
remain reachable.

#### Scenario: Project whose name is a prefix of another project's name
- **WHEN** a pane group for `git.example.com/name-two` is already open on a workspace, and
  `OpenPaneGroup` is then called for `git.example.com/name` on that same workspace
- **THEN** the existence check for `git.example.com/name` reports that it does not exist, a
  new and distinct pane group is created for it at `worktree_path`, and the pane group for
  `git.example.com/name-two` is left untouched

#### Scenario: Switching to a project whose name is a prefix of another
- **WHEN** pane groups for both `git.example.com/name` and `git.example.com/name-two` exist on
  a workspace and `SwitchTo` is called with the pane group ID for `git.example.com/name`
- **THEN** the client is switched or attached to the `git.example.com/name` pane group, not to
  `git.example.com/name-two`

#### Scenario: Creation order does not affect resolution
- **WHEN** the pane group for `git.example.com/name` is created first and `OpenPaneGroup` is
  then called for `git.example.com/name-two`
- **THEN** a distinct pane group is created for `git.example.com/name-two`, and subsequently
  switching to either pane group reaches that exact pane group

#### Scenario: Existing pane group is still reused
- **WHEN** `OpenPaneGroup` is called for a project whose pane group already exists on the
  workspace under exactly that name
- **THEN** the existing pane group is reused and no new one is created

#### Scenario: Pane-ID targets are unaffected
- **WHEN** the plugin operates on a pane using a tmux-assigned pane ID
- **THEN** the pane ID is used as-is and the operation applies to that pane

### Requirement: OpenPane starts a program in a new pane

`session-tmux` SHALL implement `Session.OpenPane({workspace_id, pane_group_id,
argv, cwd, env})` by creating a new pane in the named pane group on the named
workspace socket and returning a `Pane` whose `pane_id` is the identifier tmux
assigned to it.

- `argv` SHALL be run as the pane's program. It is an already-split argument
  vector; the plugin SHALL quote each element so the multiplexer's shell
  re-parses it into the same vector, and an element containing spaces or shell
  metacharacters SHALL NOT be split or expanded.
- An empty `argv` SHALL start the provider's default shell.
- `cwd`, when non-empty, SHALL be the pane's starting directory.
- `env` entries SHALL be set in the pane's environment. They SHALL be applied in
  a deterministic order so that repeated calls issue identical commands.
- The pane group SHALL be addressed by exact name, never by prefix or glob.

Where the new pane is placed is provider policy and is not part of the contract.

An empty `workspace_id` or `pane_group_id` SHALL be rejected with
`INVALID_ARGUMENT`. A pane group that does not exist SHALL be reported as
`NOT_FOUND`.

#### Scenario: Pane started with an argument vector

- **WHEN** `OpenPane({workspace_id: "<sock>", pane_group_id: "github•com/kalbasit/swm", argv: ["my-tool", "--flag", "two words"], cwd: "/tmp/wt"})` is called
- **THEN** a new pane is created in the `github•com/kalbasit/swm` pane group with
  starting directory `/tmp/wt`, running `my-tool --flag 'two words'`, and the
  returned `Pane` carries the identifier tmux assigned, the requested
  `pane_group_id`, and the requested `workspace_id`

#### Scenario: Empty argv starts a shell

- **WHEN** `OpenPane` is called with an empty `argv`
- **THEN** the pane is created with no explicit program and runs the default
  shell

#### Scenario: Environment is applied deterministically

- **WHEN** `OpenPane` is called with `env = {B: "2", A: "1"}`
- **THEN** the environment entries are applied in a stable order that does not
  depend on map iteration order

#### Scenario: Missing pane group

- **WHEN** `OpenPane` names a pane group that does not exist on the workspace
- **THEN** the call fails with `NOT_FOUND`

#### Scenario: Missing identifiers rejected

- **WHEN** `OpenPane` is called with an empty `workspace_id` or an empty
  `pane_group_id`
- **THEN** the call fails with `INVALID_ARGUMENT` and no tmux command is issued

### Requirement: ListPanes enumerates panes across workspaces

`session-tmux` SHALL implement `Session.ListPanes({workspace_id,
pane_group_id})` by streaming one `Pane` per live pane.

- An empty `workspace_id` SHALL enumerate every live workspace socket under the
  socket directory, applying the same liveness probe `ListWorkspaces` uses, so
  that a single call answers what is running on the host.
- A non-empty `workspace_id` SHALL restrict the results to that workspace, and
  SHALL be reported as `NOT_FOUND` when no such socket exists.
- A non-empty `pane_group_id` SHALL restrict the results to panes in that pane
  group.
- Each streamed `Pane` SHALL carry `pane_id`, `pane_group_id`, and
  `workspace_id`, plus the pane title, the command currently running in it, its
  current working directory, and whether it is focused.
- A pane is `focused` when the multiplexer reports that it is the active pane of
  the active window of a session that has at least one attached client — that
  is, when a human's keystrokes would currently reach it.
- Results SHALL be ordered deterministically.

#### Scenario: All panes on the host

- **WHEN** two workspaces are live and `ListPanes({})` is called
- **THEN** panes from both workspaces are streamed, each carrying the workspace
  socket it came from

#### Scenario: Filtered by workspace

- **WHEN** `ListPanes({workspace_id: "<sock-a>"})` is called and both `<sock-a>`
  and `<sock-b>` are live
- **THEN** only panes from `<sock-a>` are streamed

#### Scenario: Filtered by pane group

- **WHEN** `ListPanes({workspace_id: "<sock>", pane_group_id: "github•com/kalbasit/swm"})` is called
- **THEN** only panes belonging to that pane group are streamed

#### Scenario: Focus reported

- **WHEN** a pane is the active pane of the active window of an attached session
- **THEN** its streamed `Pane` has `focused = true`, and panes that no attached
  client is currently typing into have `focused = false`

#### Scenario: Unknown workspace

- **WHEN** `ListPanes` names a workspace socket that does not exist
- **THEN** the call fails with `NOT_FOUND`

#### Scenario: Dead sockets skipped

- **WHEN** a socket file exists but its tmux server is no longer running and
  `ListPanes({})` is called
- **THEN** that socket contributes no panes and the call succeeds

### Requirement: SendText delivers text to a pane as if typed

`session-tmux` SHALL implement `Session.SendText({workspace_id, pane_id, text,
submit, delay_ms, allow_focused})` by delivering `text` to the pane and, when
`submit` is set, the Enter key after it.

- `text` SHALL be delivered literally. No part of it SHALL be interpreted as a
  key name, so text such as `Enter`, `C-c`, or a leading `-n` arrives as those
  characters and not as the corresponding key or a command flag.
- `submit` SHALL append the provider's submit key as a separate delivery.
- `delay_ms`, when positive, SHALL be observed *before* delivering, matching the
  semantics of `pane_cmd_delay` in the layout spec, and SHALL abort with the
  context's error if the context is cancelled while waiting.
- An empty `text` with `submit = true` SHALL be valid and deliver only the
  submit key, so a caller that needs a gap between text and submission can issue
  two calls.
- An empty `text` with `submit = false` SHALL be rejected with
  `INVALID_ARGUMENT`; there is nothing to deliver.
- An empty `workspace_id` or `pane_id` SHALL be rejected with
  `INVALID_ARGUMENT`.
- A `pane_id` that is not present on the workspace SHALL be reported as
  `NOT_FOUND`.

#### Scenario: Text and submit

- **WHEN** `SendText({workspace_id: "<sock>", pane_id: "%4", text: "hello", submit: true})` is called for an unfocused pane
- **THEN** the literal text `hello` is delivered to pane `%4`, followed by a
  separate Enter delivery

#### Scenario: Text that looks like a key name

- **WHEN** `SendText` is called with `text = "Enter"`
- **THEN** the five characters `Enter` are delivered, not the Enter key

#### Scenario: Text with a leading dash

- **WHEN** `SendText` is called with `text = "-n oops"`
- **THEN** the text is delivered as-is and is not interpreted as a flag

#### Scenario: Submit only

- **WHEN** `SendText` is called with an empty `text` and `submit = true`
- **THEN** only the Enter key is delivered

#### Scenario: Nothing to send

- **WHEN** `SendText` is called with an empty `text` and `submit = false`
- **THEN** the call fails with `INVALID_ARGUMENT`

#### Scenario: Delay observed before delivery

- **WHEN** `SendText` is called with `delay_ms = 200`
- **THEN** the delivery is issued no earlier than 200 ms after the call begins

#### Scenario: Unknown pane

- **WHEN** `SendText` names a `pane_id` that does not exist on the workspace
- **THEN** the call fails with `NOT_FOUND` and nothing is delivered

### Requirement: SendText refuses a focused pane unless overridden

Text delivered by `SendText` is indistinguishable from typing. A pane a human is
currently typing into may be displaying a confirmation prompt, and injected text
would answer it. `session-tmux` SHALL therefore refuse delivery when the target
pane is focused, as defined in the `ListPanes` requirement, and SHALL report
`FAILED_PRECONDITION` with a message naming `allow_focused` as the override.

When `allow_focused` is set, the plugin SHALL deliver regardless of focus. The
refusal is a guard against the common accident, not a security boundary: the
caller that sets `allow_focused` owns the consequences.

The focus determination SHALL come from the same provider query that populates
`Pane.focused`, so that a caller which sees `focused = false` from `ListPanes`
and immediately calls `SendText` is not refused for a reason it could not have
observed. The check is inherently racy — a human can focus the pane between the
two calls — and callers SHALL NOT treat a successful `SendText` as proof that no
human was typing.

#### Scenario: Focused pane refused

- **WHEN** `SendText` targets a pane that an attached client is currently
  focused on, without `allow_focused`
- **THEN** the call fails with `FAILED_PRECONDITION`, the message names
  `allow_focused`, and no text is delivered

#### Scenario: Focused pane with explicit override

- **WHEN** the same call is made with `allow_focused = true`
- **THEN** the text is delivered

#### Scenario: Unattached session is not focused

- **WHEN** `SendText` targets the active pane of a workspace that has no
  attached client
- **THEN** the pane is not treated as focused and the text is delivered

### Requirement: ClosePane terminates a pane idempotently

`session-tmux` SHALL implement `Session.ClosePane({workspace_id, pane_id})` by
killing the named pane on the named workspace socket.

A pane that no longer exists SHALL NOT be an error: the call SHALL succeed, so
that a caller cleaning up after a program that already exited does not have to
distinguish the two. An empty `workspace_id` or `pane_id` SHALL be rejected with
`INVALID_ARGUMENT`.

#### Scenario: Close a live pane

- **WHEN** `ClosePane({workspace_id: "<sock>", pane_id: "%4"})` is called and the
  pane exists
- **THEN** the pane is killed and the call succeeds

#### Scenario: Close a pane that is already gone

- **WHEN** `ClosePane` names a pane that no longer exists
- **THEN** the call succeeds

#### Scenario: Missing identifiers rejected

- **WHEN** `ClosePane` is called with an empty `workspace_id` or an empty
  `pane_id`
- **THEN** the call fails with `INVALID_ARGUMENT`
