## Context

The repo currently contains only the Nix flake, openspec scaffolding, and agent/skill configuration. No Go code exists. Phase 0 lays the complete structural foundation — module graph, proto definitions, SDK skeleton, and build tooling — that every subsequent phase builds on. Getting the module graph and proto layout right here is critical; changing them later cascades into every plugin module.

## Goals / Non-Goals

**Goals:**

- Establish the Go module graph (`go.work` + per-module `go.mod` files) with correct import paths for `proto/`, `sdk/go/`, `cmd/swm/`, and the four bundled plugin stubs
- Define all six proto services in `proto/swm/plugin/v1/` with full message types (compile-only — no server implementations yet)
- Provide a Go SDK skeleton at `sdk/go/` that plugin authors will import: one package per capability with the interface types and a stub `Serve()` entry point
- Wire the Taskfile (root + per-module), golangci-lint config, and Nix devshell updates so `task fmt`, `task lint`, and `task test` all exit 0 from a clean state

**Non-Goals:**

- Any plugin business logic or host core logic
- The `swm` CLI doing anything useful beyond `swm --version`
- CI pipeline (GitHub Actions) — addressed in a later change
- Contract test harness (`sdk/go/contract/`) — scaffolded but empty; filled in Phase 1+

## Decisions

### D1: `proto/` as its own Go module

`proto/` gets `module github.com/kalbasit/swm/proto` in its `go.mod`. Generated Go code from `buf gen` is written directly into `proto/swm/plugin/v1/`. Every plugin and the host import it as a standalone dependency.

**Why not generate into `sdk/go/`?** The SDK depends on the proto, not the other way round. Keeping them as separate modules means a plugin author can import just the proto (for type sharing) without pulling in the full SDK. It also allows the proto to be versioned independently if a `v2/` proto is ever needed.

### D2: `buf` for proto code generation

`buf.yaml` at `proto/` root declares the module and lint rules. `buf.gen.yaml` drives `protoc-gen-go` + `protoc-gen-go-grpc`. The devshell gains `pkgs.buf`, `pkgs.protoc-gen-go`, `pkgs.protoc-gen-go-grpc`.

**Why `buf` over raw `protoc`?** buf handles plugin version pinning in `buf.gen.yaml`, provides breaking-change detection (`buf breaking`), and produces consistent output across machines without a shared `$PATH` of protoc plugins. Raw protoc requires every developer to have matching `protoc-gen-go` versions; buf pins them.

**`go_package` option:** `option go_package = "github.com/kalbasit/swm/proto/swm/plugin/v1;pluginv1"` in every `.proto` file. The Go package name `pluginv1` avoids collisions when multiple proto versions coexist.

### D3: SDK structure — one package per capability

`sdk/go/session/`, `sdk/go/vcs/`, `sdk/go/forge/`, `sdk/go/picker/` — each exports:
- The capability interface (what a plugin must implement)
- A `Serve(impl Interface)` function that starts the go-plugin server

The `Serve` function is a thin wrapper around `go-plugin`'s `plugin.Serve()`. In Phase 0 the interface types are defined but `Serve` is a stub that panics with "not implemented" in test builds and returns an error in production — replaced with real logic in Phase 1.

**Why interfaces in `sdk/go/`, not in `cmd/swm/`?** The host also needs these interface types to construct the client side. Placing them in the SDK means both the host and the plugin import the same types without a circular dependency. (This follows the TDD §6.3 pattern; see also openspec config: "interfaces defined in the consumer package" applies within a module boundary, not across module boundaries.)

### D4: Taskfile structure

Root `Taskfile.dist.yml` uses `version: "3"`, `set: [errexit, nounset, pipefail]`. Per-module `tasks.yml` files under `proto/`, `sdk/go/`, `cmd/swm/`, and each `plugins/<name>/`. The `proto:gen` task uses `sources:` / `generates:` for caching. The root `test` task explicitly lists modules in order: `proto:test`, `sdk:test`, `swm:test`.

In Phase 0, plugin module stubs (`plugins/vcs-git/`, `plugins/session-tmux/`, etc.) contain only `main.go` with `func main() {}` and a `go.mod`. Their `tasks.yml` files define `build`, `lint`, and `test` stubs that run successfully but do nothing useful yet. This means the root Taskfile includes are complete from day one; later phases only add content inside the module, never reshape the root.

### D5: golangci-lint configuration

`.golangci.yml` is adapted from the Stowix reference (version 2 format, gofumpt formatter). Linters removed that don't apply to a CLI+plugin project: `zerologlint` (we use `log/slog`, not zerolog), `bodyclose`/`rowserrcheck`/`sqlclosecheck` (no HTTP/DB in Phase 0). Linters added: `forbidigo` (to catch `panic` and `os.Exit` outside `main`) with patterns matching the no-panic-outside-main rule. `gochecknoglobals` is enabled (enforces no global state outside `main`).

The pre-commit golangci-lint entry is rewritten to loop over `cmd/ sdk/go/ proto/ plugins/*/` instead of `apps/*/`.

### D6: Nix devshell shellHook extended to plugins

The existing glob `cmd/**/go.mod sdk/**/go.mod` is extended to `cmd/**/go.mod sdk/**/go.mod plugins/**/go.mod` so plugin `go.mod` files get their Go version auto-updated on `direnv reload`. Without this, plugin modules would drift to a different Go version from the host after a `nix flake update`.

## Risks / Trade-offs

- **buf output is deterministic only when pinned.** If `buf.gen.yaml` pins exact plugin versions (via managed mode or explicit paths from the Nix store), `proto:gen` is idempotent. If it calls `protoc-gen-go` from `$PATH` without version pins, different machines could produce slightly different output. Mitigation: use `buf.gen.yaml` managed mode pointing to the Nix-provided binaries; commit generated code so `git diff` in CI catches drift.

- **go.work complicates `go get`/`go mod tidy`.** With five modules in one workspace, `go mod tidy` must be run per-module, not at the root. Mitigation: each `tasks.yml` includes a `tidy` task; document the workflow in `README.md`.

- **Empty plugin stubs add `golangci-lint` noise.** An `empty main()` will trigger `gochecknomainresult` or similar. Mitigation: Phase 0 plugin stubs include a trivial `fmt.Println("swm-plugin-…")` so the file is lintable without suppression directives.

## Open Questions

_(none — all decisions confirmed with repository owner prior to drafting)_
