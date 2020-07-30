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
  buildInputs = with nixpkgs; [ go gotools ];

  GOPATH = "${home}/.cache/go";
  GO111MODULE = "on";

  shellHook = builtins.concatStringsSep "\n" [
    # the shell hook is exporting an invalid cert file. Get rid of this cert
    # file if it does not exist.
    ''
      if [ -n "''${SSL_CERT_FILE:-}" ] || [ -n "''${NIX_SSL_CERT_FILE:-}" ]; then
        local ca=""
        if [ -n "''${SSL_CERT_FILE:-}" ] && [ -f "$SSL_CERT_FILE" ]; then
          ca="$SSL_CERT_FILE"
        elif [ -n "''${NIX_SSL_CERT_FILE:-}" ] && [ -f "$NIX_SSL_CERT_FILE" ]; then
          ca="$NIX_SSL_CERT_FILE"
        elif [ -f"/etc/ssl/certs/ca-bundle.crt" ]; then
          ca="/etc/ssl/certs/ca-bundle.crt"
        fi
        if [ -f "$ca" ]; then
            export SSL_CERT_FILE="$ca"
            export NIX_SSL_CERT_FILE="$ca"
          fi
      else
        unset SSL_CERT_FILE NIX_SSL_CERT_FILE
      fi
    ''
  ];
}
