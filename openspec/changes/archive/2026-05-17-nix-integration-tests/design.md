# Design: nix-integration-tests

## Context

`cmd/swm/tests/integration/` lives inside the `cmd/swm` Go module but is excluded from the
`packages.swm` Nix derivation because its `TestMain` calls `go build` on sibling modules
(`plugins/vcs-git`, `plugins/session-tmux`, `plugins/picker-fzf`, `plugins/forge-github`) at
test time. `buildGoModule` sandboxes each module in isolation with no network access, so
cross-module `go build` calls fail at evaluation time.

The result: `nix flake check` (the sole CI gate) never runs the integration suite.

All five plugin binaries already exist as Nix packages (`packages.swm-plugin-*`). The
test helpers `faketmux` and `fakefzf` are small programs inside the `session-tmux` and
`picker-fzf` modules respectively.

## Goals / Non-Goals

**Goals:**
- `nix flake check` exercises the full integration test suite.
- No network access required at test time (tests already use in-process mocks for GitHub).
- Local `go test ./tests/integration/...` continues to work without Nix (falls back to
  `go build`).
- `packages.swm` is unchanged (integration tests remain excluded from that derivation).

**Non-Goals:**
- Running integration tests inside `packages.swm`'s check phase.
- Testing with a real tmux socket or a real fzf binary.
- Parallelism between unit tests and integration tests within a single derivation.

## Decisions

### Decision 1: Inject pre-built binaries via environment variables

`TestMain` in `main_test.go` will check a set of env vars before calling `go build`:

| Binary          | Env var                     |
|-----------------|-----------------------------|
| vcs-git plugin  | `SWM_PLUGIN_VCS_GIT_BIN`    |
| session-tmux plugin | `SWM_PLUGIN_SESSION_TMUX_BIN` |
| picker-fzf plugin | `SWM_PLUGIN_PICKER_FZF_BIN` |
| forge-github plugin | `SWM_PLUGIN_FORGE_GITHUB_BIN` |
| faketmux        | `SWM_TEST_FAKETMUX_BIN`     |
| fakefzf         | `SWM_TEST_FAKEFZF_BIN`      |

When an env var is set the variable is assigned directly; the `go build` call is skipped
for that binary. When unset, the existing `go build` path runs unchanged (local dev).

**Alternative considered:** Always pass binaries on the `PATH` and have tests discover them
by name. Rejected because it requires renaming `faketmux` to `tmux` on the path, which is
fragile and can interfere with the actual tmux the host has installed.

### Decision 2: Separate Nix derivations for faketmux and fakefzf

`faketmux` lives in `plugins/session-tmux/internal/session/testdata/faketmux` (inside the
`session-tmux` module). `fakefzf` lives in `plugins/picker-fzf/internal/picker/testdata/fakefzf`
(inside the `picker-fzf` module). Since they belong to separate modules they cannot be built
inside a `cmd/swm`-rooted `buildGoModule`. Two new derivations are created:

- `packages.swm-test-faketmux`: `modRoot = "plugins/session-tmux"`,
  `subPackages = [ "internal/session/testdata/faketmux" ]`
- `packages.swm-test-fakefzf`: `modRoot = "plugins/picker-fzf"`,
  `subPackages = [ "internal/picker/testdata/fakefzf" ]`

Both share the same source fileset pattern as their parent plugin packages.

**Alternative considered:** Build faketmux/fakefzf in `preCheck` of the integration
derivation using `go build`. Rejected: the Nix sandbox has no network, so the Go toolchain
cannot resolve the `session-tmux` module's dependencies from within `cmd/swm`'s vendor tree.

**Alternative considered:** Install faketmux/fakefzf as outputs of the existing plugin
packages. Rejected: test helper binaries pollute production package outputs.

### Decision 3: New `checks.swm-integration-tests` derivation, not a package

A dedicated check entry keeps CI failures easy to diagnose (`checks.swm-integration-tests`
fails independently of `packages.swm`). The derivation uses `modRoot = "cmd/swm"` and
the **same** `vendorHash` as `packages.swm` — the `go.mod`/`go.sum` are identical; only the
source fileset differs (integration tests are included, not excluded).

Build phase is skipped (`subPackages = []`); only the check phase runs
`go test ./tests/integration/...`. The install phase emits an empty `$out` directory.

### Decision 4: nativeBuildInputs

The integration derivation requires `pkgs.git` (the vcs-git plugin invokes `git` via
`os/exec`). No other external tools are needed: tmux is replaced by `faketmux`, fzf by
`fakefzf`, and GitHub is mocked in-process with `net/http/httptest`.

## Risks / Trade-offs

**[Risk] vendorHash drift** — `checks.swm-integration-tests` duplicates the `vendorHash`
from `packages.swm`. If `go.mod` changes, both must be updated together.
→ Mitigation: a comment in the check derivation points to `packages.swm` as the source of
truth; CI will catch hash mismatches at build time.

**[Risk] faketmux/fakefzf vendorHash duplication** — same issue: these derivations share
`vendorHash` with their parent plugin packages.
→ Same mitigation: cross-reference comments.

**[Risk] Integration tests depend on internal packages** — `tests/integration` imports
`cmd/swm/internal/...`. If those packages gain external dependencies not already in the
`cmd/swm` vendor tree, the hash must be updated.
→ Acceptable: already the case for `packages.swm`; no new risk introduced.

## Migration Plan

1. Add `packages.swm-test-faketmux` and `packages.swm-test-fakefzf` to
   `nix/packages/flake-module.nix` (new import files under `nix/packages/`).
2. Modify `cmd/swm/tests/integration/main_test.go`: env-var injection before each
   `buildBin` call.
3. Add `checks.swm-integration-tests` in `nix/checks/flake-module.nix`.
4. Update `openspec/specs/nix-packages/spec.md`: replace the "integration tests are absent"
   scenario with the new requirement.
5. Run `nix flake check` locally to verify the new check passes before pushing.

No rollback needed: changes are additive. Removing the new check derivation reverts to the
current state.

## Open Questions

- None. Approach is fully determined by the constraint that `buildGoModule` is single-module.
