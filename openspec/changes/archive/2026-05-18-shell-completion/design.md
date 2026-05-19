## Context

`swm` is a Cobra-based CLI. Cobra ships with a built-in `completion` subcommand
(and `GenBashCompletionV2`, `GenZshCompletion`, `GenFishCompletion`, etc.) that
is registered on the root command by default in Cobra v1.8+. Currently the root
command does not opt out, so the subcommand should already be present — but no
completions are installed into the Nix package.

The Nix package (`nix/packages/swm/default.nix`) uses `buildGoModule`. It does
not currently use `installShellFiles` and has no `postInstall` hook.

## Goals / Non-Goals

**Goals:**
- Expose `swm completion bash|zsh|fish|powershell` as a working subcommand.
- Install completion scripts for bash, zsh, and fish into the Nix profile via
  `installShellFiles`.

**Non-Goals:**
- Dynamic argument completion (story names, workspace names, repo slugs) — that
  requires `ValidArgsFunction` on individual subcommands and is a follow-on.
- PowerShell installation via Nix (no `installShellCompletion` support for it).
- Completion for plugin subcommands beyond what Cobra generates statically.

## Decisions

### 1. Use Cobra's built-in completion — no custom completion logic

**Decision:** rely entirely on Cobra's auto-generated `completion` subcommand.

Cobra v1.8 auto-registers `completion` unless `root.CompletionOptions.DisableDefaultCmd = true`.
The current root command does not set this option, so the subcommand is already
present. No Go code changes are needed for basic completion.

If `swm completion bash` is tested and found absent, the fix is a single line:
`root.InitDefaultCompletionCmd()` (or removing any `DisableDefaultCmd` setting).

**Alternative considered:** write a custom `completion` subcommand that calls
the individual `Gen*` methods. Rejected — Cobra's built-in is functionally
identical and eliminates maintenance burden.

### 2. Generate and install completions in Nix `postInstall`

**Decision:** add `installShellFiles` to `nativeBuildInputs` and a `postInstall`
hook that runs the built binary to generate scripts, then installs them.

```nix
nativeBuildInputs = [ pkgs.git pkgs.installShellFiles ];

postInstall = ''
  installShellCompletion --cmd swm \
    --bash <($out/bin/swm completion bash) \
    --zsh  <($out/bin/swm completion zsh)  \
    --fish <($out/bin/swm completion fish)
'';
```

`installShellCompletion` from `installShellFiles` places scripts at the
conventional paths (`share/bash-completion/completions/`, `share/zsh/site-functions/`,
`share/fish/vendor_completions.d/`) so that Nix profiles wire them up
automatically.

**Alternative considered:** commit static completion scripts to the repo and
copy them at build time. Rejected — scripts would diverge from the command tree
as `swm` evolves; generating at build time is always in sync.

## Risks / Trade-offs

- [Risk] `postInstall` runs the binary on the build host — fails if binary is
  not host-executable (e.g. cross-compilation). → Mitigation: `swm` is only
  packaged for native targets today; if cross-compilation is added later, guard
  `postInstall` with `stdenv.hostPlatform == stdenv.buildPlatform`.

- [Trade-off] Static completion (no dynamic arg completion). Users get
  subcommand and flag completion but not story/workspace name completion until a
  follow-on change adds `ValidArgsFunction` to the relevant subcommands.
