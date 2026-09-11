## Why

`swm --version` prints `v2.0.0-dev` on every build, whatever it was built from:
`cmd/swm/main.go` hardcodes the string and nothing ever overwrites it. Builds
from `v2.6.0`, `v2.7.0` and `v2.7.1` are indistinguishable in the field, so the
only way to tell which swm a host runs is to probe for a subcommand. The Nix
packages already compute the right value (`version.txt` tag, else the flake
revision) but only use it to name the derivation — the `ldflags` wiring that
would carry it into the binary was never added, even though
`openspec/specs/nix-packages/spec.md` already requires that "the built binary
reports version `v1.2.3`". The four plugin packages have the same gap: their
`buildVersion` vars are commented "set via ldflags at link time" and report
`dev` instead.

## What Changes

- Add `cmd/swm/internal/version`, which resolves the version a binary reports
  from, in order: the link-time value, the Go module version recorded by
  `go install <module>@<tag>`, the VCS revision recorded by `go build` (marked
  dirty when the tree was modified), and finally `devel`.
- `cmd/swm/main.go` sets `root.Version` from that package and drops the
  hardcoded `v2.0.0-dev`.
- All five Nix packages pass their computed version through `ldflags`, so the
  derivation's version and the binary's version can no longer disagree. The
  `swm` package asserts `swm --version` matches at build time.
- The release workflow writes the pushed tag into
  `nix/packages/swm/version.txt` before building, mirroring the shared
  `kalbasit/gh-actions` build workflow, so a release build reports its tag
  without anyone editing a checked-in file.

## Capabilities

### New Capabilities
- `version-reporting`: how the `swm` binary decides which version string to
  report, across Nix builds, `go install`, and local `go build`.

### Modified Capabilities
- `nix-packages`: the computed version must reach the built binary via
  `ldflags`, not only the derivation name.
- `github-ci-cd`: the release workflow must inject the pushed tag into
  `version.txt` before building.

## Impact

- Capability surface: none. This is host CLI plus packaging; no proto changes,
  so no `proto/swm/plugin/vN/` bump.
- Code: `cmd/swm/main.go`, new `cmd/swm/internal/version`, all five
  `nix/packages/*/default.nix`, `.github/workflows/releases.yml`.
- Every package's derivation gains an `ldflags` attribute, so all five rebuild
  once. `vendorHash` values are untouched — no `proto/` file changes.
- Plugin `Info()` responses start reporting a real version instead of `dev`;
  nothing consumes that field for behaviour, so this is observable but not
  breaking.

## Non-goals

- Teaching Nix to discover the tag of an arbitrary checkout. A flake's `self`
  exposes only `rev`/`shortRev` and no ref, so a build from
  `github:kalbasit/swm/v2.7.1` reports that tag's commit SHA unless the builder
  injects the tag. That is why the release workflow does the injecting.
- Independent version tags for plugin binaries (`plugins/<name>/vX.Y.Z`).
- Any new CLI surface beyond the existing `--version` flag.
