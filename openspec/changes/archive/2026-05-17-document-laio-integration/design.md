## Context

The laio tmux layout manager was integrated into swm's session-tmux plugin (archived change
`2026-05-18-laio-plugin-support`). The integration added `{{tmux_socket}}` as a
`pane_group_command` template variable and specified `--socket` / `LAIO_TMUX_SOCKET` as the
mechanism for directing laio at swm's per-story tmux socket. None of this is visible to users
via documentation: the session-tmux README's template-variable table ends at `{{project_id}}`,
no worked example for laio exists, and the `examples/` directory is absent.

This change is documentation-only: no code, no proto, no tests.

## Goals / Non-Goals

**Goals**

- Document `{{tmux_socket}}` alongside the existing template variables.
- Show both integration patterns (per-project config and global config) with ready-to-copy snippets.
- Provide a minimal but realistic `laio.yaml` sample that works with swm's socket model.

**Non-Goals**

- Changing any runtime behaviour.
- Documenting laio itself (defer to laio's own docs for yaml semantics beyond what swm-specific notes require).
- Adding laio docs to the root README or host CLI README.

## Decisions

### D1: Location of the laio.yaml sample — `plugins/session-tmux/examples/laio.yaml`

An `examples/` subdirectory under the plugin keeps the sample co-located with the plugin README
that references it, and avoids polluting the repo root or `docs/`. The file is purely
informational and not included in any build.

**Alternatives considered:**
- `docs/examples/laio.yaml` — separates the sample from the plugin README; cross-linking is
  awkward.
- Inline in the README as a fenced code block only — loses discoverability; users cannot
  copy-paste a real file path into `--file`.

### D2: Content of `laio.yaml` sample — show `path:` variable + `force_new_windows: true`

The sample must demonstrate the two non-obvious constraints:
1. `path: "{{ path }}"` — laio's own Tera variable that swm passes via `--var path={{worktree_path}}`.
2. No `attach: true` or `switch-client` in the config — laio runs detached inside an already-attached
   session, so any attach attempt is a no-op or error. The sample omits `attach` to use laio's
   default (no-attach when `--skip-attach` is passed).

### D3: Template-variable table placement — extend existing table in session-tmux README

The existing `Configuration` section already has a `pane_group_command` row. Adding a new
"Template variables" sub-section immediately after the config table is cleaner than scattering
variable docs across the README.

### D4: Two patterns in the README, not one

Both the per-project pattern (`{{worktree_path}}/.swm/laio.yaml`) and the global pattern
(`~/.config/swm/laio.yaml` + `--var path=...`) are needed because they differ in a non-obvious
way: the per-project pattern requires `path: .` in laio.yaml (laio resolves `.` relative to
the config file, which co-locates with the project), while the global pattern requires
`path: "{{ path }}"` with an explicit `--var` injection.

## Risks / Trade-offs

- **laio version compatibility**: `--socket` and `--skip-attach` are flags from the laio version
  used in the integration. If laio renames them in a future release, the documentation will be
  wrong. Mitigation: note the minimum laio version in the README.
- **`force_new_windows` semantics**: the sample uses `force_new_windows: true` to match the
  in-session flow; omitting it causes laio to try to reuse the initial window in a way that
  leaves an extra unnamed window. This is a laio internals detail, not a swm concern, but the
  sample must get it right.

## Open Questions

_(none)_
