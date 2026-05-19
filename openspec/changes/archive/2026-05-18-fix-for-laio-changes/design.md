## Context

laio-cli's `start` command previously treated `--tmux-socket` from inside an active tmux session as an implicit signal to reconfigure the current session in-place. Commit `8727e02` in laio-cli gates that behaviour behind a new, explicit `--replace-current-session` flag. Without the flag, laio now falls back to its default path: create a new named session or switch to one. This silently breaks every swm `pane_group_command` that relies on laio patching the session swm just attached to.

No swm code changed — the existing `pane_group_command` template expansion is already correct. The fix is purely textual: add `--replace-current-session` to the canonical command string wherever it appears (docs, spec examples, tests, and the user's live config).

## Goals / Non-Goals

**Goals:**
- Every documented or tested laio invocation includes `--replace-current-session`.
- `~/.config/swm/config.toml` gains an explicit `pane_group_command` entry so the user's live environment works with the updated laio.
- The spec scenario for `OpenPaneGroup` with a custom `pane_group_command` reflects the new required flag.

**Non-Goals:**
- No changes to swm's template-expansion logic — it already handles arbitrary flags.
- No changes to the plugin gRPC surface or proto definitions.
- No version bump; this is a documentation and config correction.

## Decisions

**Single canonical command string, updated in place.**
The command `laio start --file … --tmux-socket … --skip-attach` gains one flag: `--replace-current-session`. It is inserted after `--tmux-socket` and before `--skip-attach` to match laio's own documented examples and keep flags logically grouped (socket opts, then behaviour opts).

No alternative was considered; the only question was whether to introduce a `replace_current_session` toggle in swm's TOML schema. That would add complexity with no benefit — `pane_group_command` is already a free-form string that users control, so the flag belongs there.

**`~/.config/swm/config.toml` gets the full command.**
The live config currently has no `pane_group_command`. It needs one so the user's environment actually works. The shared-layout variant (single `laio.yaml` at `~/.config/swm/laio.yaml` with `--var path=`) is the right choice because the user already has a `laio.yaml` there.

## Risks / Trade-offs

- **Older laio versions**: `--replace-current-session` is unrecognised on laio < `8727e02`. Users on older builds will get a hard error. → Acceptable; the change is tied to the specific laio version that introduced the flag.
- **Test constant**: `testLaioPaneGroupCommandTOML` and the `wantCmd` assertion are string literals. They will fail if not updated atomically with the docs. → The tasks artifact ensures both are updated together and `task test` gates the PR.
