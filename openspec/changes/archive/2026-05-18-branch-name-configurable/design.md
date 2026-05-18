## Context

`swm story create <name>` defaults the branch to `"feat/" + name` (hardcoded in
`cmd/swm/internal/cli/story/create.go:27`). There is a `--branch` override but no
project-wide or user-wide way to change the default pattern. Teams following
different conventions (e.g. `fix/`, `wael/`, `WM-123/`) must supply `--branch`
on every invocation.

`Config` (in `cmd/swm/internal/config/config.go`) currently has no story-scoped
settings. Configuration is loaded from `$XDG_CONFIG_HOME/swm/config.toml` via
`pelletier/go-toml/v2` and the host passes a `*Config` down to subcommands via
`root.go`.

## Goals / Non-Goals

**Goals:**
- Allow users to set a default branch-name pattern in `config.toml`.
- Support arbitrary templates — not just prefix-style — so `{{.Name}}` can appear
  anywhere in the pattern.
- Preserve exact backward compatibility: no config → same `feat/<name>` default.
- Fail fast with a clear error when the template is syntactically invalid or
  produces an empty string.

**Non-Goals:**
- Template variables beyond `.Name` (timestamps, git user, etc.).
- Per-project or per-story template overrides.
- Proto / plugin-protocol changes.

## Decisions

### 1. Use `text/template` (stdlib) — not `strings.ReplaceAll` or `fmt.Sprintf`

**Decision**: Use `text/template` with a single data struct `{ Name string }`.

**Alternatives considered**:
- `strings.ReplaceAll(tpl, "{name}", name)` — simpler but invents a custom
  syntax users must learn; no tooling support.
- `fmt.Sprintf`-style `%s` — breaks if the template contains a literal `%`.
- External template library (e.g. Handlebars) — unnecessary dep for a simple feature.

`text/template` is stdlib, has well-known `{{.Name}}` syntax, and compiling
the template at config-load time gives instant syntax errors.

### 2. Introduce a `[story]` TOML sub-section inside `Config`

**Decision**: Add a `Story` struct with `BranchNameTemplate string` to `Config`.

```toml
[story]
branch_name_template = "feat/{{.Name}}"
```

**Alternatives considered**:
- Top-level `branch_name_template` key — pollutes the root namespace; hard to
  group future story settings.
- Inside `[plugins]` — wrong abstraction; this is a host concern, not a plugin concern.

### 3. Parse template per call in `branchFromTemplate`, not at config-load time

**Decision**: Parse the template string inside `branchFromTemplate` on each
invocation rather than once at config-load or command-construction time.

**Alternatives considered**:
- Parse at config load (`load.go`) — requires storing a `*template.Template` in
  `Config`, which is a Go-specific type and complicates the config struct.
- Parse in `NewCreateCmd` — `NewCreateCmd` returns `*cobra.Command` (no error
  return), so a parse failure would have to be deferred to `RunE` anyway via a
  captured closure error; gains nothing over parsing in `branchFromTemplate`.

Parsing per-call keeps `branchFromTemplate` a pure, easily-testable function
with no external state. For a CLI binary the allocation overhead is negligible —
`swm story create` is invoked once per user action.

### 4. Default value: `"feat/{{.Name}}"`

This evaluates identically to the current hardcoded `"feat/" + name`, so no
existing behaviour changes for users without a config file.

## Risks / Trade-offs

- [Template injection] Config is user-supplied; a malformed template can produce
  unexpected branch names → Mitigation: validate template produces a non-empty
  string; run git-safe character checks are out of scope (branch validation is
  the VCS plugin's responsibility).
- [New struct field in Config] Downstream code that constructs `Config` directly
  in tests needs no changes because the zero value of `Story{}` will use the
  `Defaults()` path → no test breakage expected.
- [TOML key addition] Existing config files without `[story]` silently use the
  default — no migration needed.

## Migration Plan

1. Add `Story` struct and `BranchNameTemplate` field to `Config`; update
   `Defaults()` to set `"feat/{{.Name}}"`.
2. Teach `NewCreateCmd` to accept the compiled template (or the raw string) and
   evaluate it instead of `"feat/" + name`.
3. Wire the config value through `root.go` → `story.NewCreateCmd`.
4. No rollback needed: the field is optional and defaults to the current value.

## Open Questions

_(none — all decisions resolved above)_
