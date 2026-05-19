## ADDED Requirements

### Requirement: Completion subcommand

`swm` MUST expose a `completion` subcommand that accepts a shell name
(`bash`, `zsh`, `fish`, `powershell`) and writes a valid completion script
for that shell to stdout.

#### Scenario: Bash completion script

- **WHEN** the user runs `swm completion bash`
- **THEN** a valid bash completion script is printed to stdout and exits 0

#### Scenario: Zsh completion script

- **WHEN** the user runs `swm completion zsh`
- **THEN** a valid zsh completion script is printed to stdout and exits 0

#### Scenario: Fish completion script

- **WHEN** the user runs `swm completion fish`
- **THEN** a valid fish completion script is printed to stdout and exits 0

#### Scenario: PowerShell completion script

- **WHEN** the user runs `swm completion powershell`
- **THEN** a valid PowerShell completion script is printed to stdout and exits 0

#### Scenario: Completion covers all top-level subcommands

- **WHEN** the bash completion script is sourced and the user types `swm <TAB>`
- **THEN** all registered top-level subcommands appear as candidates
  (`story`, `workspace`, `pr`, `clone`, `completion`)

### Requirement: Nix shell completion installation

The `swm` Nix package MUST install shell completion scripts for bash, zsh,
and fish into the conventional locations so that Nix profiles wire them up
automatically.

#### Scenario: Bash completion installed

- **WHEN** the `swm` Nix package is built
- **THEN** a bash completion file for `swm` is present under
  `$out/share/bash-completion/completions/`

#### Scenario: Zsh completion installed

- **WHEN** the `swm` Nix package is built
- **THEN** a zsh completion file (`_swm`) is present under
  `$out/share/zsh/site-functions/`

#### Scenario: Fish completion installed

- **WHEN** the `swm` Nix package is built
- **THEN** a fish completion file for `swm` is present under
  `$out/share/fish/vendor_completions.d/`
