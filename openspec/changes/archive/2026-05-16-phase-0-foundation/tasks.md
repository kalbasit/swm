## 1. Nix Devshell and Tooling

- [x] 1.1 Add `pkgs.buf`, `pkgs.protoc-gen-go`, `pkgs.protoc-gen-go-grpc` to `nix/devshells/flake-module.nix` buildInputs
- [x] 1.2 Extend the shellHook Go-version rewrite loop in `nix/devshells/flake-module.nix` to include `plugins/**/go.mod` alongside `cmd/**/go.mod sdk/**/go.mod`
- [x] 1.3 Fix the golangci-lint pre-commit entry in `nix/pre-commit/flake-module.nix` to iterate `cmd/ sdk/go/ proto/ plugins/*/` instead of `apps/*/`
- [x] 1.4 Run `nix fmt` and verify `nix build .#checks.x86_64-linux.pre-commit-check` passes (or equivalent check target)

## 2. golangci-lint Configuration

- [x] 2.1 Create `.golangci.yml` adapted from the Stowix reference: version 2 format, gofumpt formatter, remove HTTP/DB-specific linters (`zerologlint`, `bodyclose`, `rowserrcheck`, `sqlclosecheck`), add `forbidigo` with patterns for `panic`, `os.Exit`, `log.Fatal` outside main, enable `gochecknoglobals`
- [x] 2.2 Add per-module exclusion paths so generated proto code (`proto/swm/plugin/v1/*.pb.go`, `*.pb.grpc.go`) is excluded from linting

## 3. Proto Module

- [x] 3.1 Create `proto/go.mod` with module path `github.com/kalbasit/swm/proto`; add deps `google.golang.org/grpc`, `google.golang.org/protobuf` (`proto` module)
- [x] 3.2 Create `proto/buf.yaml` (v2) declaring the module and enabling standard lint rules (BASIC + FILE_SAME_GO_PACKAGE)
- [x] 3.3 Create `proto/buf.gen.yaml` pointing at Nix-provided `protoc-gen-go` and `protoc-gen-go-grpc`, output to `.` (within proto module), managed mode off
- [x] 3.4 Write `proto/swm/plugin/v1/common.proto`: `ProjectID`, `Empty`, `BoolValue`, `PathResponse`, `Story`, `Project`, `PluginInfo`, `Capability`, `CapabilityDep` messages; `option go_package` set
- [x] 3.5 Write `proto/swm/plugin/v1/session.proto`: `Session` service with all seven RPCs from TDD §6.3; imports `common.proto`
- [x] 3.6 Write `proto/swm/plugin/v1/vcs.proto`: `VCS` service with six RPCs from TDD §6.3; imports `common.proto`
- [x] 3.7 Write `proto/swm/plugin/v1/forge.proto`: `Forge` service with four RPCs from TDD §6.3; imports `common.proto`
- [x] 3.8 Write `proto/swm/plugin/v1/picker.proto`: `Picker` service with `Info` and bidirectional-streaming `Pick` RPCs; imports `common.proto`
- [x] 3.9 Write `proto/swm/plugin/v1/host.proto`: `Host` service with six RPCs from TDD §6.4; imports `common.proto`
- [x] 3.10 Run `task proto:gen` (after Taskfile exists in step 5) to generate Go code; commit generated files
- [x] 3.11 Create `proto/tasks.yml` with `build` (`buf build`), `lint` (`buf lint`), `gen` (with sources/generates caching), and `test` (empty — no test code) tasks

## 4. SDK Go Module

- [x] 4.1 Create `sdk/go/go.mod` with module path `github.com/kalbasit/swm/sdk/go`; add deps `github.com/kalbasit/swm/proto`, `github.com/hashicorp/go-plugin`
- [x] 4.2 Create `sdk/go/handshake/handshake.go`: export `HandshakeConfig` with `MagicCookieKey = "SWM_PLUGIN_MAGIC_COOKIE"`, `MagicCookieValue`, `ProtocolVersion = 1` (`sdk/go` module)
- [x] 4.3 Create `sdk/go/session/plugin.go`: define `Plugin` interface matching the Session service methods; `Serve(Plugin) error` stub (`sdk/go` module)
- [x] 4.4 Create `sdk/go/vcs/plugin.go`: define `Plugin` interface matching the VCS service methods; `Serve(Plugin) error` stub (`sdk/go` module)
- [x] 4.5 Create `sdk/go/forge/plugin.go`: define `Plugin` interface matching the Forge service methods; `Serve(Plugin) error` stub (`sdk/go` module)
- [x] 4.6 Create `sdk/go/picker/plugin.go`: define `Plugin` interface matching the Picker service methods; `Serve(Plugin) error` stub (`sdk/go` module)
- [x] 4.7 Create `sdk/go/tasks.yml` with `build` (`go build ./...`), `lint` (`golangci-lint run ./...`), and `test` (`go test ./...`) tasks

## 5. cmd/swm Stub

- [x] 5.1 Create `cmd/swm/go.mod` with module path `github.com/kalbasit/swm/cmd/swm`; add deps `github.com/spf13/cobra`, `github.com/kalbasit/swm/sdk/go`
- [x] 5.2 Create `cmd/swm/main.go`: cobra root command with `--version` flag only; no subcommands yet
- [x] 5.3 Create `cmd/swm/tasks.yml` with `build` (`go build -o swm ./`), `lint`, and `test` tasks

## 6. Plugin Stubs

- [x] 6.1 Create `plugins/vcs-git/go.mod` (`github.com/kalbasit/swm/plugins/vcs-git`), `main.go` with trivial `fmt.Println("swm-plugin-vcs-git")` body, and `tasks.yml` with `build` target producing `swm-plugin-vcs-git` binary (`plugins/vcs-git` module)
- [x] 6.2 Create `plugins/session-tmux/go.mod` (`github.com/kalbasit/swm/plugins/session-tmux`), `main.go` stub, and `tasks.yml` producing `swm-plugin-session-tmux` (`plugins/session-tmux` module)
- [x] 6.3 Create `plugins/picker-fzf/go.mod` (`github.com/kalbasit/swm/plugins/picker-fzf`), `main.go` stub, and `tasks.yml` producing `swm-plugin-picker-fzf` (`plugins/picker-fzf` module)
- [x] 6.4 Create `plugins/forge-github/go.mod` (`github.com/kalbasit/swm/plugins/forge-github`), `main.go` stub, and `tasks.yml` producing `swm-plugin-forge-github` (`plugins/forge-github` module)

## 7. go.work and Root Taskfile

- [x] 7.1 Create `go.work` at repo root referencing all six modules: `proto/`, `sdk/go/`, `cmd/swm/`, `plugins/vcs-git/`, `plugins/session-tmux/`, `plugins/picker-fzf/`, `plugins/forge-github/`
- [x] 7.2 Create root `Taskfile.dist.yml`: `version: "3"`, `set: [errexit, nounset, pipefail]`, `includes:` for all seven module taskfiles, `default` → `task --list`, `fmt` → `nix fmt`, `lint` → `deps:` on all module lint tasks, `test` → sequential `task:` entries in order (proto, sdk, swm, vcs-git, session-tmux, picker-fzf, forge-github)
- [x] 7.3 Run `task fmt` → confirm exits 0
- [x] 7.4 Run `task lint` → confirm exits 0
- [x] 7.5 Run `task test` → confirm exits 0
