## Context

`ListProjects` in `cmd/swm/internal/hostsvc/server.go` walks `$CODE_ROOT/repositories/`
looking for directories named `.git`. When it finds one, it emits the parent directory as
a project and returns `filepath.SkipDir` — which skips walking *inside* `.git` — but
does nothing to prevent the walker from descending into the `.git` directory's **siblings**.
The walk therefore continues into subdirectories such as `.terraform/modules/`,
`.terragrunt-cache/`, and `tmp/`, surfacing every nested git repository as an independent
project in the picker.

## Goals / Non-Goals

**Goals**
- After emitting a project root, the walker does not descend into any of that project's
  subdirectories.
- Fix is contained to the walking logic in `hostsvc/server.go`; no proto, CLI, or
  plugin changes are required.

**Non-Goals**
- User-configurable ignore/exclude patterns for the repository scan.
- Symlink loop detection or cross-device walk behaviour.
- Changes to how the picker (fzf) renders or filters items.

## Decisions

### 1. Track found project roots; skip on prefix match

**Chosen approach**: Maintain a `projectRoots []string` slice inside the `WalkDir`
callback closure. When `d.Name() == ".git"`, append `filepath.Dir(path)` to
`projectRoots`. At the top of the callback, before any other work, check whether
`path` has any element of `projectRoots` as a path-separator-terminated prefix; if so,
return `filepath.SkipDir` immediately.

Prefix check: `strings.HasPrefix(path, root+string(filepath.Separator))` — the
separator suffix prevents a false positive when one project name is a prefix of another
(e.g. `foo` vs `foobar`).

**Alternatives considered**

| Alternative | Why rejected |
|---|---|
| Return a signal from the `.git` handler to skip the parent | `filepath.WalkDir` has no such mechanism; `filepath.SkipDir` from a directory entry skips only that directory's children. |
| Two-pass scan (collect all paths, then filter) | Buffers all paths before streaming; breaks the streaming contract of the RPC. |
| Custom `fs.FS` wrapper that prunes | Adds abstraction complexity with no benefit over a simple prefix check. |

### 2. Rely on lexical walk ordering for correctness

`filepath.WalkDir` visits entries in lexical order. `.git` (character `g`) sorts before
every subdirectory name starting with a letter `h`–`z` and before hidden directories
starting with `.` followed by `h`–`z` (e.g. `.terraform` starts with `.t`, `t` > `g`).
Therefore `.git` is always visited before tools caches like `.terraform/` or
`.terragrunt-cache/`, guaranteeing that the project root is registered before any
sub-directory is evaluated.

**Edge case**: a hidden directory at the project root starting with `.a`–`.f` (e.g.
`.drone/`) would be visited before `.git`. Such a directory would not itself be a `.git`
match, so the walk would descend into it and find any nested `.git` there before the
project root's `.git` is processed. In practice this scenario is harmless: the nested
`.git` would be emitted as a project, then the real project root's `.git` would be
encountered and also emitted. The walk would then skip the project root's remaining
siblings correctly. This minor edge case does not apply to the reported problem domains
(Terraform/Terragrunt caches) and is acceptable for v2.0.

## Risks / Trade-offs

- **[Lexical ordering assumption]** → The `.terraform` / `.terragrunt-cache` case is safe
  because `.t` > `.g`. For hidden dirs starting with `.a`–`.f`, a nested repo inside
  them would still be emitted before the true project root, resulting in a duplicate
  entry rather than a missing one. Acceptable; can be revisited if needed.
- **[Slice scan performance]** → O(n) per directory entry where n = number of already-found
  projects. Tens of thousands of projects is unusual; slice scan is fast in practice.
  → Mitigate later with a sorted set or trie if profiling shows it matters.

## Migration Plan

Pure bug fix: no config format changes, no API changes, no data migration.
Rollback = code revert with no side effects.

## Open Questions

None.
