# Update Nix vendorHashes When Proto Files Change

Whenever any file under `proto/` is modified — including formatting-only changes
to `.pb.go` files — run:

```sh
task update-nix-vendor-hashes
```

then commit the resulting nix changes alongside the proto changes.

## Why

All five Go packages vendor the local `proto` module via a `replace` directive:

```
replace github.com/kalbasit/swm/proto => ../../proto
```

`buildGoModule` runs `go mod vendor`, which copies the local proto tree verbatim
into the vendor directory. Any content change in `proto/` — even a blank line —
produces a different vendor hash, invalidating the stored `vendorHash` in:

- `nix/packages/swm/default.nix`
- `nix/packages/swm-plugin-forge-github/default.nix`
- `nix/packages/swm-plugin-picker-fzf/default.nix`
- `nix/packages/swm-plugin-session-tmux/default.nix`
- `nix/packages/swm-plugin-vcs-git/default.nix`

## When this fires

- After `task fmt` if proto files appear in the diff
- After running `protoc` or any proto-regeneration step
- After any manual edit to a file under `proto/`

## What the task does

Mirrors the `generate` job in `.github/workflows/ci.yml`: for each package it
attempts to build `.#${pkg}.goModules`, extracts the `got:` hash from the error
output if the stored hash is wrong, and patches the `vendorHash` line in place
using Python.
