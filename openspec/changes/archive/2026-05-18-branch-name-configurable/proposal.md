## Why

The default branch name for new stories is hardcoded to `feat/<name>` in
`cmd/swm/internal/cli/story/create.go`. Teams that follow different branch
naming conventions (e.g. `fix/<name>`, `wael/<name>`, or `WM-123/<name>`)
must pass `--branch` on every `swm story create` invocation — there is no
way to configure a project-wide or user-wide default template.

## What Changes

- Add a `branch_name_template` field to the `[story]` TOML section of
  `$XDG_CONFIG_HOME/swm/config.toml` that accepts a Go `text/template`
  string with access to the story name (`.Name`).
- Default value: `feat/{{.Name}}` (backward-compatible — existing behaviour
  is preserved when no config is set).
- `swm story create` evaluates the template to derive the default branch
  name; the `--branch` flag still overrides it unconditionally.
- Template rendering errors surface as a clear user-facing error before any
  store write occurs.

## Non-goals

- Changing how the branch name is stored or used after creation (no proto
  changes; `BranchName` in the story JSON is still a plain string).
- Dynamic tokens beyond story name (e.g. timestamps, git user) — out of
  scope for v2.0.
- Per-project or per-story override of the template.

## Capabilities

### New Capabilities

_(none)_

### Modified Capabilities

- **`workflow-commands`** — the default branch name derivation in
  `swm story create` changes from a hardcoded `"feat/"+name` string to a
  configurable template evaluated from `Config.Story.BranchNameTemplate`.
  A delta spec is required.

## Impact

- `cmd/swm/internal/config/config.go` — add `Story` sub-struct with
  `BranchNameTemplate string` field; update `Defaults()` to set
  `"feat/{{.Name}}"`.
- `cmd/swm/internal/cli/story/create.go` — replace `"feat/" + name` with
  template evaluation; propagate config via `root.go`.
- No plugin protocol (proto) changes.
- No forge/session/vcs/picker/hook capability surfaces affected.
