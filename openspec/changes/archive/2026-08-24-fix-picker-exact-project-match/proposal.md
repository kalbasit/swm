## Why

When two repos on the same host share a name prefix — e.g. `git.entreprise.com/name` and
`git.entreprise.com/name-two` — selecting `name` in the project picker opens `name-two`
instead. The picker and the host-side key resolution are both exact; the defect is that the
`session-tmux` plugin passes session and window names to `tmux -t` unescaped. tmux resolves a
`-t` target by trying an exact match, then a **prefix match**, then fnmatch, then substring.
With a `name-two` session already running, `has-session -t <...>/name` prefix-matches it and
succeeds, so swm never creates the `name` session, and the subsequent `switch-client` /
`attach-session` resolves to `name-two` for the same reason.

The user silently lands in the wrong repo with no error — the failure is invisible until they
notice the working directory is wrong, and it is data-losing in the sense that edits get made
in the wrong worktree.

## What Changes

- Escape every tmux `-t` target in the `session-tmux` plugin with tmux's exact-match prefix
  `=`, so a target name is matched literally and never prefix/fnmatch/substring-matched:
  - `session/tmux.go`: `has-session`, `switch-client`, `attach-session`.
  - `layout/apply.go`: `setenv`, `rename-window`, `new-window`, and the `display-message`
    used to resolve `#{pane_id}`.
  - Pane-ID targets (`%N` from `display-message -p '#{pane_id}'`) are already unambiguous and
    stay as-is: `select-pane`, `resize-pane`, `send-keys`, `kill-pane`.
- `-s` on `new-session` is a literal name, not a target, and is left unchanged.
- Teach the `faketmux` test double to track created session names and emulate tmux's real
  target-resolution order (exact → prefix → fnmatch), so the shared-prefix case is
  reproducible in a unit test. Today `has-session` in the fake is a flat `FAKETMUX_HAS_SESSION`
  env toggle, which is precisely why no existing test catches this.
- Add regression coverage: with a `<host>/name-two` pane group already open, opening
  `<host>/name` must create a distinct session and attach to it.
- Update the two existing argv assertions in `tmux_test.go` that expect an unescaped
  `attach-session -t <name>`.

No **BREAKING** changes: session names on disk are unchanged, only how they are *addressed*.
Existing running sessions keep working.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `session-tmux`: session and window targeting becomes exact-match. Adds the requirement that
  pane-group creation, lookup, and attach address tmux by literal name, so a project whose
  name is a prefix of another project's name resolves to its own session.

## Impact

**Affected code**

- `plugins/session-tmux/internal/session/tmux.go` — `OpenPaneGroup` (`has-session`),
  `SwitchTo` (`switch-client`, `attach-session`).
- `plugins/session-tmux/internal/layout/apply.go` — `setenv`, `rename-window`, `new-window`,
  `display-message`.
- `plugins/session-tmux/internal/session/testdata/faketmux/main.go` — needs stateful session
  tracking with tmux-like resolution.
- `plugins/session-tmux/internal/session/tmux_test.go` — argv assertions.

**Not affected** (verified exact, no change needed)

- `picker-fzf`: `fzf.go` returns the selected line's key field verbatim.
- `cmd/swm/internal/cli/workspace/open.go`: `buildCandidates`, `projectIDFromKey`, and
  `isAttached` use exact string equality and explicit splits.
- `cmd/swm/internal/core/layout/layout.go`: its one `strings.HasPrefix` is against
  `<code_root>/repositories/` with a trailing separator, so it cannot confuse the two names.

**Capability surface**: session.

**Proto**: no changes — this is entirely plugin-internal, so no version bump under
`proto/swm/plugin/vN/`.

**Dependencies**: none. The `=` exact-match target prefix is long-standing tmux syntax.
