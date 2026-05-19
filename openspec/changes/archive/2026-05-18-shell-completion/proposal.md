## Why

`swm` has no shell completion, making subcommands, flags, and arguments
undiscoverable via tab-completion in bash, zsh, or fish. Cobra provides this
for free once wired up; it only needs to be activated and installed.

## What Changes

- Add Cobra's built-in `completion` subcommand to the `swm` root command so
  users can run `swm completion bash|zsh|fish|powershell`.
- Add `installShellFiles` to the `nix/packages/swm/default.nix` package and
  generate completion scripts at build time so they are installed into the
  Nix profile alongside the binary.

## Capabilities

### New Capabilities

- `shell-completion` — the `swm completion` subcommand and Nix-side
  installation of generated completion scripts for bash, zsh, and fish.

### Modified Capabilities

<!-- none -->

## Impact

- `cmd/swm/internal/cli/root.go` — enable Cobra completion; no proto changes.
- `nix/packages/swm/default.nix` — add `installShellFiles` native build input
  and a `postInstall` hook that generates and installs the completion scripts.
