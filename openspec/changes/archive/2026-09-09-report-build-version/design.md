## Context

See proposal.md — Why. The constraint that shapes everything here is what Nix
can and cannot know. Verified on this repo:

```
$ nix eval --impure --json --expr \
    'builtins.attrNames (builtins.getFlake "github:kalbasit/swm/v2.7.1").sourceInfo'
["lastModified","lastModifiedDate","narHash","outPath","rev","shortRev"]
```

A flake's `self` exposes no ref and no tag, even when the flake was fetched by
tag. The build sandbox has no `.git` either, so `git describe` is not an option.
A tag can therefore only reach a build in one of two ways: it is committed to
the source tree, or the builder injects it. soxincfg — the only consumer that
matters today — pins `github:kalbasit/swm` by revision and consumes
`inputs.swm.packages.<system>.swm-full`, so it takes whatever the swm
derivation produces and needs no change.

`ncps` solves the same problem with the shared `kalbasit/gh-actions` build
workflow, which writes `github.ref_name` into `nix/packages/<pkg>/version.txt`
at build time (never committing it) and passes the result through
`-X ...Version=${version}`. swm's `releases.yml` is standalone and does neither.

## Goals / Non-Goals

**Goals:**

- One source of truth per build: the derivation's version attribute and the
  string the binary prints are the same value, and a build fails if they drift.
- A binary built outside Nix (`go build`, `go install`) still reports something
  truthful, without any build-system cooperation.

**Non-Goals:**

- Changing how the derivation's version attribute itself is computed. Steps 1–3
  of the existing `nix-packages` requirement stay exactly as they are.
- Making an arbitrary `nix build github:kalbasit/swm/v2.7.1#swm` report `v2.7.1`.
  That is not reachable without committing the tag; such a build reports the
  tag's commit SHA, which still identifies it exactly.

## Decisions

**A dedicated `cmd/swm/internal/version` package, not a `var` in `main`.**
`main` cannot be unit-tested, and the resolution order has four branches worth
testing. The package exposes `Resolve(linked string, info *debug.BuildInfo)` —
a pure function over its inputs, so tests drive it directly — plus `Version()`,
which applies it to the link-time variable and `debug.ReadBuildInfo()`.
Alternative considered: `-X main.version=`, matching the current shape. Rejected
because it leaves the logic untestable and the ldflag path depends on the main
package's import path, which is easy to get wrong silently.

**The link-time variable defaults to empty, not to a version string.** The
current `v2.0.0-dev` default is the bug: it is release-shaped, so a missing
injection is invisible. An empty default makes a missing injection fall through
to the Go build-info fallbacks, which produce a `devel`-prefixed string.

**Fall back to `debug.BuildInfo` before giving up.** `go install <mod>@v1.2.3`
records the module version and `go build` in a work tree records
`vcs.revision`/`vcs.modified`. Reading them costs nothing and covers every
non-Nix build path, including a developer's `go build ./cmd/swm` in the
devshell.

**Nix passes the version verbatim; Go prints what it is given.** The alternative
— having Nix synthesise a display string like `0.0.0-dev+<rev>` — would change
the derivation's version attribute, and therefore every store path name, for no
gain: a bare 40-character SHA is already impossible to mistake for a release.

**The `swm` package asserts `swm --version` in `postInstall`.** This is the only
check that closes the actual regression: the wiring existing but not reaching
the binary. It reuses the existing `hostPlatform == buildPlatform` guard that
already wraps the shell-completion generation, which runs `$out/bin/swm` the
same way and in the same shell where `preCheck` has already exported `HOME` and
`XDG_RUNTIME_DIR`. Alternative considered: a separate flake check. Rejected as
strictly more machinery for the same assertion.

**Plugins get the same ldflag, targeting their existing `buildVersion` vars.**
Those vars already exist and are already documented as ldflag-injected; the
injection was simply never written. Leaving four packages lying is worse than
the one-line-per-package cost of fixing them.

## Risks / Trade-offs

- **A build from a tag still reports a SHA unless CI injects the tag** →
  Documented as a non-goal above and mitigated where it matters: the release
  workflow injects `github.ref_name`, and every other build reports a revision
  that maps back to a commit unambiguously.
- **`postInstall` now depends on `swm --version` starting cleanly in the
  sandbox** → The same phase already runs `$out/bin/swm completion bash`, which
  goes through the identical `main()` path, so this adds no new environmental
  requirement.
- **All five derivations change, so everything rebuilds once** → Unavoidable and
  one-time; `vendorHash` values are untouched because no `proto/` file changes.
- **Plugin `Info()` version strings change from `dev` to a real version** →
  Nothing branches on that field; it is reported, not consumed.
