let
  hostPkgs = import <nixpkgs> {};
  # Look here for information about how to generate `nixpkgs-version.json`.
  #  → https://nixos.wiki/wiki/FAQ/Pinning_Nixpkgs
  pinnedVersion = hostPkgs.lib.importJSON ./.nixpkgs-version.json;
  pinnedPkgs = import (hostPkgs.fetchFromGitHub {
    owner = "NixOS";
    repo = "nixpkgs";
    inherit (pinnedVersion) rev sha256;
  }) {};
in

# This allows overriding nixpkgs by passing `--arg nixpkgs ...`
{ nixpkgs ? pinnedPkgs, home ? builtins.getEnv "HOME" }:

nixpkgs.mkShell {
  buildInputs = with nixpkgs; [
    go
    dep
  ];

  GOPATH = "${home}/.cache/go";
  GO111MODULE = "on";
}
