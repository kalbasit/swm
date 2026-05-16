#!/usr/bin/env bash

cd -- "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"

nix develop --command ./scripts/.worktree-init.sh
