## Why

`ListProjects` in the host service walks `repositories/` looking for `.git` directories but
does not stop descending into a project once its root `.git` is found, so sub-repositories
(e.g. Terraform module caches under `.terraform/modules/`, terragrunt caches, temporary
clone directories) are surfaced alongside real top-level repositories in the fzf picker.

## What Changes

- `cmd/swm/internal/hostsvc/server.go` — `ListProjects`: after emitting a project whose
  root contains a `.git` marker, skip all further descent into that project directory so
  nested `.git` directories are never visited.
- No proto changes. No new RPCs. No configuration surface.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- **`vcs-git`** — add a new requirement: once the host discovers a `project_markers` entry
  (`.git`) inside a directory, it SHALL NOT descend further into that directory's
  subdirectories when scanning for additional projects.  This prevents sub-repositories
  (tool caches, vendored modules, temporary clones) from being surfaced as independent
  projects.

## Impact

- `cmd/swm/internal/hostsvc/server.go` — `ListProjects` walk logic (≈ 10 lines).
- `cmd/swm/internal/hostsvc/server_test.go` — new table-driven test cases covering the
  nested-repo scenario.
- `openspec/specs/vcs-git/spec.md` — one new requirement section.
- No API, proto, CLI, or plugin changes required.

## Non-goals

- Configurable ignore/exclude patterns for the repository scan.
- Handling symlinks that point outside the repository tree.
- Any changes to how the picker plugin (fzf) renders or filters items.
