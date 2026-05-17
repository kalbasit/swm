## Context

The repository is being prepared for public release. There are currently no `LICENSE` or `README` files anywhere in the monorepo. The monorepo contains: a host CLI (`cmd/swm`), a Go SDK (`sdk/go`), and four plugins (`forge-github`, `picker-fzf`, `session-tmux`, `vcs-git`). Each is a separate Go module in a `go.work` workspace.

## Goals / Non-Goals

**Goals:**
- Add an Apache 2.0 `LICENSE` at the repository root.
- Provide a high-level `README.md` at the root for first-time visitors.
- Provide focused `README.md` files in `cmd/swm/`, `sdk/go/`, and each plugin directory.

**Non-Goals:**
- License header injection into source files.
- CI license compliance checks.
- Auto-generated API docs or a documentation site.
- READMEs for internal packages (e.g. `pkg/`, `proto/`).

## Decisions

### 1 — Apache 2.0 as the license
Chosen by the project owner. It is permissive, widely understood, and compatible with the project's dependencies.

### 2 — Two-tier README structure
Root README is intentionally high-level (30-second pitch, architecture summary, links). Module READMEs go deeper (configuration, CLI flags, plugin-specific behaviour). This avoids a single enormous README while keeping each entry-point self-contained.

### 3 — Consistent plugin README template
All four plugin READMEs follow the same section order: Purpose → Requirements → Configuration → Usage → Limitations. Consistency matters more than per-plugin creativity since users will read multiple plugins.

### 4 — No Badges / CI shields in initial pass
Shields require stable release tags and a working CI badge URL. Adding them now would produce broken images. They can be added once v2 cuts a release.

## Risks / Trade-offs

- **Documentation drift** — READMEs will fall out of sync with code as the project evolves. Mitigation: keep content factual and minimal; prefer linking to code/config rather than duplicating it.
- **Incomplete quick-start** — Until a release binary is published, install instructions are necessarily from source. Mitigation: document `go install` and Nix paths clearly; note that pre-built binaries are coming.
