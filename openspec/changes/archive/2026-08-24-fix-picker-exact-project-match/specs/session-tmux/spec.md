## ADDED Requirements

### Requirement: Exact-match tmux target resolution
`session-tmux` SHALL address every existing tmux session and window by literal name, so that
a target name is never resolved to a different session or window whose name merely starts
with, matches a glob against, or contains the requested name.

tmux resolves an unescaped `-t` target by trying, in order: exact name, prefix match, fnmatch,
then substring. Because pane-group names are derived from project IDs, a project whose name is
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

## MODIFIED Requirements

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
