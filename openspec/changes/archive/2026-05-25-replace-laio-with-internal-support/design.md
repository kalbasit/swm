## Context

`session-tmux` currently delegates window/pane layout to Laio via `pane_group_command`:

```
pane_group_command = "laio start --file '{{worktree_path}}/.swm/laio.yaml' --tmux-socket '{{tmux_socket}}' --replace-current-session --skip-attach"
```

Laio is an external Rust binary that reads a YAML layout config, connects to the given tmux socket, replaces the bare session with a fully-laid-out one (named windows, nested pane splits, startup commands), and exits. It is the only piece SWM cannot influence and the only reason users must install a second tool.

The `OpenPaneGroup` RPC already receives everything needed — `workspace_id` (the socket path), `worktree_path`, `project_id`, `story_name` — to apply a layout without external help.

## Goals / Non-Goals

**Goals:**
- `OpenPaneGroup` resolves a layout config from a two-tier lookup (global then per-repo, per-repo wins) and applies the window/pane layout using direct tmux commands.
- Global layout at `$XDG_CONFIG_HOME/swm/session-tmux.toml` acts as a user-wide default for every project that has no per-repo config — the same tier model as hooks.
- Per-repo layout at `.swm/session-tmux.toml` inside the worktree overrides the global config for that repository.
- Layout supports: named windows with per-window `path`, nested pane splits with `flex`/`flex_direction`, per-pane `commands`, `env`, `shell`, `focus`, `zoom`, and per-session `startup` commands.
- Template variable substitution (`worktree_path`, `story_name`, `tmux_socket`) via `text/template`.
- No config file at either tier → existing default behavior (editor + shell). `pane_group_command` still takes priority for users who haven't migrated.
- Zero new Go module dependencies.

**Non-Goals:**
- Zellij support (out of scope for v2.0).
- Serializing an existing tmux session back to TOML.
- Array template variables (Laio's `--var k=v` repeated); flat string map only in this pass.
- Changing the gRPC protocol or proto definitions.
- Migrating users automatically — the opt-in is adding `.swm/session-tmux.toml`.

## Decisions

### 1. Layout logic lives inside `session-tmux` (no new plugin, no proto changes)

**Chosen**: New sub-package `plugins/session-tmux/internal/layout/` called from `OpenPaneGroup`. No gRPC surface changes.

**Alternatives considered**:
- *New `session-layout` capability plugin* — adds a new RPC surface, deployment complexity, and inter-plugin coordination. Rejected: over-engineering for what is tmux-specific logic.
- *Host-side layout via new proto RPC* — would require proto version bump and host changes. Rejected: host should not know tmux window geometry.

### 2. Config format is TOML, not YAML

**Chosen**: `.swm/session-tmux.toml` — consistent with SWM's entire config stack (pelletier/go-toml/v2 already in the workspace module graph).

**Alternatives considered**:
- *Retain Laio's YAML format* — allows copy-paste migration, but would require a new YAML dependency (gopkg.in/yaml.v3) and creates inconsistency in the project config story. Rejected.
- *JSON* — machine-friendly but poor for humans writing layout configs. Rejected.

### 3. Two-tier config resolution: global then per-repo (per-repo wins)

**Chosen**: Mirror the hook resolution model. `OpenPaneGroup` probes in this order and uses the first match:

```
1. pane_group_command in config.toml           (backward compat — wins if set)
2. <worktree_path>/.swm/session-tmux.toml      (per-repo — overrides global)
3. $XDG_CONFIG_HOME/swm/session-tmux.toml      (global default)
4. built-in default (editor + shell)
```

Unlike hooks (which are composing — all tiers run), layout is an override chain — only the first matching tier applies. Two configs cannot meaningfully be merged into a single window layout.

The global config uses the same TOML schema as the per-repo config. Template variables (`worktree_path`, `story_name`, `tmux_socket`) are injected into both tiers identically, so a single global layout can adapt to each project's paths.

**Alternatives considered**:
- *Compose tiers (merge windows from global + per-repo)* — ambiguous semantics (duplicate window names, ordering). Rejected.
- *Per-repo only* — forces every project to have a config file; no sensible user-wide default. Rejected.
- *Per-story tier* (like hooks have) — layout is a project property, not a story property; the same repo should look the same regardless of which story checked it out. Rejected for this surface.

### 4. Flex-split algorithm: recursive percentage split

Panes form a recursive tree (`Pane.Panes []Pane`). At each node:

```
totalFlex = Σ flex(child)
For i = 0; i < len(children)-1; i++:
    remainingFlex = Σ flex(children[i+1:])
    splitPercent  = remainingFlex * 100 / (remainingFlex + flex(children[i]))
    tmux split-window -p <splitPercent> [-h if row direction]
```

This keeps children[i] at `flex[i]/total` of the parent and recurses into the remainder. Rounding errors accumulate to the last pane (same as Laio's behavior).

**Alternatives considered**:
- *Absolute size in cells* — more precise, but terminal size at creation time is unknown and may change. Rejected.
- *Flat percentage in config* — defeats the purpose of flex; manual and error-prone. Rejected.

### 5. `pane_group_command` takes precedence over layout file

If `pane_group_command` is set in `config.toml`, `OpenPaneGroup` behaves exactly as today (no layout file read). This preserves backward compatibility for anyone already using a custom command.

Migration path: remove `pane_group_command` from `config.toml` and add `.swm/session-tmux.toml`.

### 6. Template engine: `text/template` (stdlib)

Variables `worktree_path`, `story_name`, `tmux_socket` are injected. Syntax matches the existing `pane_group_command` template format already in `tmux.go` — consistent, zero new dependencies.

**Alternative**: Sprig (rich function library) — over-kill for three string substitutions. Rejected.

### 7. Layout is applied only on session creation (idempotent)

`OpenPaneGroup` is already idempotent: if the tmux session exists it returns immediately. Layout is therefore applied exactly once per session lifetime. No state file is written.

## Risks / Trade-offs

- **[Risk] Flex rounding produces non-pixel-perfect splits** → Acceptable: Laio has the same constraint. Last pane absorbs rounding error. Document clearly.
- **[Risk] Layout config references a relative `path` that doesn't exist in the worktree** → Mitigation: validate paths during config load; return a descriptive error before touching tmux.
- **[Risk] User has both `pane_group_command` and a layout config file (either tier)** → The command silently wins. Mitigation: log a warning naming the ignored file.
- **[Risk] Global config uses a path like `./src` that only makes sense for some repos** → Template variables (`worktree_path`) let users write absolute paths or expressions; document that relative paths in global config resolve from `worktree_path`.
- **[Risk] Startup commands in config produce stderr that corrupts the pane** → Identical risk in Laio; user responsibility. No mitigation needed.
- **[Risk] `split-window -p` argument rounds to 0 or 100 for extreme flex ratios** → Mitigation: clamp to [1, 99] and document.

## Migration Plan

1. Ship the layout engine in `session-tmux` with zero default impact.
2. Users who want to migrate: copy `.swm/laio.yaml` content into `.swm/session-tmux.toml` (format is structurally identical, keys map 1:1; only difference is YAML→TOML syntax).
3. Remove `pane_group_command` from `config.toml` to activate the internal layout.
4. Rollback: re-add `pane_group_command` — the old Laio path still works as long as the binary is installed.
5. Once satisfied: uninstall Laio and delete the old `.swm/laio.yaml`.

## Open Questions

*(none)*

## Resolved Decisions

- **No `shutdown` commands in layout config** — SWM's hooks system (`close-workspace.d/`) already handles teardown at the right moment. Adding a second shutdown mechanism in the layout config would create two parallel teardown systems with no clear guidance on which to use. Users who need teardown logic use hooks.
- **`zoom` and `focus` are creation-time only** — the layout config describes initial state, not a persistent constraint. `SwitchTo` does not re-apply them. Users can freely change focus and zoom after opening without swm fighting them.
