## Why

swm v1 is tightly coupled to tmux, git, and GitHub with no extension points. Building swm v2 as a plugin-host architecture (see TDD §1) requires a monorepo skeleton, versioned proto definitions for all capability surfaces, a Go SDK that hides go-plugin boilerplate, and the CI/tooling infrastructure to develop, lint, and test every module. Phase 0 creates that foundation — nothing in Phases 1–4 can land without it.

## What Changes

- New top-level monorepo layout: `proto/`, `sdk/go/`, `cmd/swm/`, `plugins/` (empty stubs), `go.work` wiring them together (see TDD §4)
- Protobuf definitions for all six capability services under `proto/swm/plugin/v1/`: `common.proto`, `session.proto`, `vcs.proto`, `forge.proto`, `picker.proto`, `host.proto` — compile-only, no gRPC implementations yet
- `proto/` as its own Go module (`github.com/kalbasit/swm/proto`); generated Go code lives inside it; `buf.yaml` + `buf.gen.yaml` drive code generation via `pkgs.buf`
- Go SDK skeleton at `sdk/go/`: one sub-package per capability (`session/`, `vcs/`, `forge/`, `picker/`), each exporting the interface types and a stub `Serve()` helper — no business logic yet
- Root `Taskfile.dist.yml` with per-module includes (`proto`, `sdk`, `swm`, per-plugin); `fmt` → `nix fmt`; `lint` → per-module `golangci-lint`; `test` → sequential per-module runs (see TDD §10 Phase 0)
- `.golangci.yml` adapted from the repo reference, scoped to Go quality linters appropriate for a CLI+plugin project
- `nix/devshells/flake-module.nix`: add `pkgs.buf`, `pkgs.protoc-gen-go`, `pkgs.protoc-gen-go-grpc`; extend shellHook Go-version rewrite loop to include `plugins/**/go.mod`
- `nix/pre-commit/flake-module.nix`: fix golangci-lint entry to iterate the v2 module dirs (`cmd/`, `sdk/go/`, `proto/`, `plugins/*/`) instead of the old `apps/*/`

## Capabilities

### New Capabilities

- `plugin-protocol`: The versioned gRPC/proto contract all plugins speak — proto files, buf config, generated Go stubs, and the `go_package` import path (`github.com/kalbasit/swm/proto/swm/plugin/v1`)
- `sdk-go`: The Go SDK surface plugin authors use — per-capability interface types, `Serve()` helpers, and the go-plugin handshake constants

### Modified Capabilities

_(none — this is a greenfield change)_

## Non-goals

- No plugin implementations (git, tmux, fzf, github) — those are Phase 1+
- No CLI commands beyond `swm --version` (cobra wired up but empty)
- No host core logic (story store, config loader, plugin manager) — Phase 1
- No CI pipeline changes beyond updating the pre-commit hook entries

## Impact

- Affects: `nix/devshells/flake-module.nix`, `nix/pre-commit/flake-module.nix`, new `proto/`, `sdk/go/`, `cmd/swm/`, `Taskfile.dist.yml`, `.golangci.yml`
- Proto changes: introduces `proto/swm/plugin/v1/` (new, no version bump needed — this is v1 genesis)
- Downstream: all future phases import `github.com/kalbasit/swm/proto` and `github.com/kalbasit/swm/sdk/go`
