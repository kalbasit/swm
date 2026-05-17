## 1. License

- [x] 1.1 Add `LICENSE` file at repo root with full Apache License 2.0 text, copyright Wael Nasreddine 2026

## 2. Root README

- [x] 2.1 Write `README.md` at repo root: project name, one-line description, use-case summary
- [x] 2.2 Add architecture overview section: host + plugin model, five capability surfaces
- [x] 2.3 Add quick-start section: install from source (`go install`), Nix path, basic config
- [x] 2.4 Add plugin discovery and hook system summary with links to `cmd/swm/README.md`
- [x] 2.5 Add contributing section and links to per-module READMEs

## 3. Host CLI README

- [x] 3.1 Write `cmd/swm/README.md`: available commands and flags
- [x] 3.2 Document TOML configuration format with example `config.toml`
- [x] 3.3 Document plugin discovery order (config paths → XDG → PATH)
- [x] 3.4 Document hook system: tiers, resolution order, event names

## 4. Go SDK README

- [x] 4.1 Write `sdk/go/README.md`: purpose and target audience (plugin authors)
- [x] 4.2 Document how to implement each capability interface with minimal example
- [x] 4.3 Document how to register a plugin (call `plugin.Serve`)
- [x] 4.4 Document how to declare required/optional capability dependencies in `Info()`

## 5. Plugin READMEs

- [x] 5.1 Write `plugins/forge-github/README.md`: purpose, GitHub token requirements, config keys, example config
- [x] 5.2 Write `plugins/picker-fzf/README.md`: purpose, fzf binary requirement, config keys, example config
- [x] 5.3 Write `plugins/session-tmux/README.md`: purpose, tmux version requirement, config keys (socket path, etc.), example config
- [x] 5.4 Write `plugins/vcs-git/README.md`: purpose, git binary requirement, config keys, example config
